package template

import (
	"bytes"
	"errors"
	"fmt"
	"gnd.la/app/profile"
	"gnd.la/html"
	itemplate "gnd.la/internal/template"
	"gnd.la/internal/templateutil"
	"gnd.la/loaders"
	"gnd.la/log"
	"gnd.la/template/assets"
	"gnd.la/util/pathutil"
	"gnd.la/util/stringutil"
	"html/template"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template/parse"
)

type FuncMap map[string]interface{}

func (f FuncMap) asTemplateFuncs() map[string]interface{} {
	fns := make(map[string]interface{})
	for k, v := range f {
		if k != "" && k[0] == '#' {
			k = k[1:]
		}
		fns[k] = v
	}
	return fns
}

func (f FuncMap) add(name string, value interface{}) {
	if name != "" {
		if name[0] == '#' {
			// delete non-pure if exists
			delete(f, name[1:])
		} else {
			// delete pure if exists
			delete(f, "#"+name)
		}
		f[name] = value
	}
}

func (f FuncMap) has(name string) bool {
	return f[name] != nil || (name != "" && ((name[0] == '#' && f[name[1:]] != nil) || (name[0] != '#' && f["#"+name] != nil)))
}

type VarMap map[string]interface{}

func (v VarMap) unpack(ns string) VarMap {
	var key string
	var rem string
	if p := strings.SplitN(ns, ".", 2); len(p) > 1 {
		key = p[0]
		rem = p[1]
	} else {
		key = ns
	}
	var ret VarMap
	val := v[key]
	switch x := val.(type) {
	case VarMap:
		ret = x
	case map[string]interface{}:
		ret = VarMap(x)
	default:
		rv := reflect.ValueOf(val)
		if rv.IsValid() && rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
			m := make(map[string]interface{}, rv.Len())
			for _, k := range rv.MapKeys() {
				m[k.String()] = rv.MapIndex(k).Interface()
			}
			ret = VarMap(m)
		}
	}
	if ret != nil && rem != "" {
		return ret.unpack(rem)
	}
	return ret
}

const (
	leftDelim             = "{{"
	rightDelim            = "}}"
	dataKey               = "Data"
	varsKey               = "Vars"
	topBoilerplateName    = "_gondola_top_hooks"
	bottomBoilerplateName = "_gondola_bottom_hooks"
	topAssetsFuncName     = "_gondola_topAssets"
	AssetFuncName         = "asset"
	bottomAssetsFuncName  = "_gondola_bottomAssets"
	topBoilerplate        = "{{ _gondola_topAssets }}"
	bottomBoilerplate     = "{{ _gondola_bottomAssets }}"
	nsSep                 = "."
	nsMark                = "|"
)

var (
	ErrNoAssetsManager       = errors.New("template does not have an assets manager")
	ErrAssetsAlreadyPrepared = errors.New("assets have been already prepared")
	commentRe                = regexp.MustCompile(`(?s:\{\{\\*(.*?)\*/\}\})`)
	keyRe                    = regexp.MustCompile(`(?s:\s*([\w\-_])+?(:|\|))`)
	defineRe                 = regexp.MustCompile(`(\{\{\s*?define.*?\}\})`)
	blockRe                  = regexp.MustCompile(`\{\{\s*block\s*("[\w\-_]+")\s*?(.*?)\s*?\}\}`)
	topTree                  = compileTree(topBoilerplate)
	bottomTree               = compileTree(bottomBoilerplate)
	templatePrepend          = fmt.Sprintf("{{ $%s := .%s }}\n", varsKey, varsKey)
)

type Hook struct {
	Template *Template
	Position assets.Position
}

type Template struct {
	AssetsManager *assets.Manager
	Minify        bool
	namespace     []string
	tmpl          *itemplate.Template
	prog          *program
	name          string
	Debug         bool
	loader        loaders.Loader
	trees         map[string]*parse.Tree
	texts         map[string]string
	final         bool
	funcMap       FuncMap
	vars          VarMap
	root          string
	assetGroups   []*assets.Group
	topAssets     []byte
	bottomAssets  []byte
	contentType   string
	hooks         []*Hook
	children      []*Template
	loaded        []string
}

func (t *Template) init() {
	t.tmpl = itemplate.New("")
	// This is required so text/template calls t.init()
	// and initializes the common data structure
	t.tmpl.New("")
	funcs := FuncMap{
		topAssetsFuncName:    nop,
		bottomAssetsFuncName: nop,
		"#" + AssetFuncName:  t.Asset,
	}
	t.Funcs(funcs).Funcs(templateFuncs)
}

func (t *Template) rebuild() error {
	// Since text/template won't let us remove nor replace a parse
	// tree, we have to create a new html/template from scratch
	// and add the trees we have.
	t.init()
	for k, v := range t.trees {
		if err := t.AddParseTree(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (t *Template) Name() string {
	return t.name
}

func (t *Template) IsFinal() bool {
	return t.final
}

func (t *Template) Root() string {
	return t.root
}

func (t *Template) Trees() map[string]*parse.Tree {
	return t.trees
}

func (t *Template) Namespace() string {
	return strings.Join(t.namespace, nsSep)
}

func (t *Template) AddNamespace(ns string) {
	t.namespace = append(t.namespace, ns)
}

func (t *Template) Asset(arg string) (string, error) {
	if t.AssetsManager != nil {
		return t.AssetsManager.URL(arg), nil
	}
	return "", ErrNoAssetsManager
}

func (t *Template) Assets() []*assets.Group {
	return t.assetGroups
}

func (t *Template) AddAssets(groups []*assets.Group) error {
	if err := t.noCompiled("can't add assets"); err != nil {
		return err
	}
	t.assetGroups = append(t.assetGroups, groups...)
	return nil
}

func (t *Template) InsertTemplate(tmpl *Template, name string) error {
	if err := t.noCompiled("can't insert template"); err != nil {
		return err
	}
	if err := t.importTrees(tmpl, name); err != nil {
		return err
	}
	t.children = append(t.children, tmpl)
	return nil
}

func (t *Template) Hook(hook *Hook) error {
	if err := t.noCompiled("can't add hook"); err != nil {
		return err
	}
	for _, h := range hook.Template.hooks {
		if err := t.Hook(h); err != nil {
			return err
		}
	}
	if err := t.importTrees(hook.Template, ""); err != nil {
		return err
	}
	for k, v := range hook.Template.funcMap {
		if t.funcMap == nil {
			t.funcMap = make(FuncMap)
		}
		if !t.funcMap.has(k) {
			t.funcMap.add(k, v)
		}
	}
	t.hooks = append(t.hooks, hook)
	return nil
}

func (t *Template) SetDropComments(drop bool) error {
	if err := t.noCompiled("can't change wheter to drop comments"); err != nil {
		return err
	}
	t.tmpl.DropComments = drop
	return nil
}

func (t *Template) Compile() error {
	if err := t.noCompiled("can't compile"); err != nil {
		return err
	}
	if err := t.prepareHooks(); err != nil {
		return err
	}
	for _, v := range t.referencedTemplates() {
		if _, ok := t.trees[v]; !ok {
			log.Debugf("adding missing template %q as empty", v)
			tree := compileTree("")
			t.AddParseTree(v, tree)
		}
	}
	if err := t.prepareAssets(); err != nil {
		return err
	}
	prog, err := newProgram(t)
	if err != nil {
		return err
	}
	t.prog = prog
	return nil
}

func (t *Template) noCompiled(msg string) error {
	if t.prog != nil {
		return fmt.Errorf("%s, template is already compiled", msg)
	}
	return nil
}

func (t *Template) namespaceIn(parent *Template) string {
	from := 0
	if parent != nil {
		from = len(parent.namespace)
	}
	return strings.Join(t.namespace[from:], nsSep)
}

func (t *Template) qname(name string) string {
	return NamespacedName(t.namespace, name)
}

func (t *Template) importTrees(tmpl *Template, name string) error {
	for k, v := range tmpl.trees {
		if k == topBoilerplateName || k == bottomBoilerplateName {
			continue
		}
		var treeName string
		if k == tmpl.root && name != "" {
			treeName = name
		} else {
			treeName = tmpl.qname(k)
		}
		if _, ok := t.trees[treeName]; ok {
			return fmt.Errorf("duplicate template %q", k)
		}
		if err := t.AddParseTree(treeName, namespacedTree(v, tmpl.namespace)); err != nil {
			return err
		}
	}
	return nil
}

func (t *Template) preparedAssetsGroups(vars VarMap, parent *Template, groups [][]*assets.Group) ([][]*assets.Group, error) {
	for _, v := range t.assetGroups {
		if (t.Debug && v.Options.NoDebug()) || (!t.Debug && v.Options.Debug()) {
			// Asset enabled only for debug or non-debug
			continue
		}
		if len(v.Assets) == 0 {
			continue
		}
		if v.Options.Bundle() && v.Options.Cdn() {
			return nil, fmt.Errorf("asset group %s has incompatible options \"bundle\" and \"cdn\"", v.Names())
		}
		// Make a copy of the group, so assets get executed and compiled, every
		// time the template is loaded. This is specially useful while developing
		// a Gondola app which uses compilable or executable assets.
		v = copyGroup(v)
		// Check if any assets have to be compiled (LESS, CoffeScript, etc...)
		for _, a := range v.Assets {
			if a.IsTemplate() {
				name, err := executeAsset(t, parent, vars, v.Manager, a)
				if err != nil {
					return nil, err
				}
				a.Name = name
			}
			name, err := assets.Compile(v.Manager, a.Name, a.Type, v.Options)
			if err != nil {
				return nil, fmt.Errorf("error compiling asset %q: %s", a.Name, err)
			}
			a.Name = name
		}
		added := false
		// Don't add bundable groups if we're not going to bundle. Otherwise, it
		// messes with asset ordering.
		if !t.Debug && v.Options.Bundable() {
			for ii, g := range groups {
				if g[0].Options.Bundable() || g[0].Options.Bundle() {
					if canBundle(g[0], v) {
						added = true
						groups[ii] = append(groups[ii], v)
						break
					}
				}
			}
		}
		if !added {
			groups = append(groups, []*assets.Group{v})
		}
	}
	var err error
	for _, v := range t.hooks {
		if groups, err = v.Template.preparedAssetsGroups(vars, parent, groups); err != nil {
			return nil, err
		}
	}
	for _, v := range t.children {
		if groups, err = v.preparedAssetsGroups(vars, parent, groups); err != nil {
			return nil, err
		}
	}
	return groups, nil
}

type groupsByPriority []*assets.Group

func (g groupsByPriority) Len() int {
	return len(g)
}

func (g groupsByPriority) Less(i, j int) bool {
	v1, v2 := 0, 0
	var err error
	if g[i].Options != nil {
		v1, err = g[i].Options.Priority()
		if err != nil {
			panic(fmt.Errorf("invalid priority in group %v: %s", g[i], err))
		}
	}
	if g[j].Options != nil {
		v2, err = g[j].Options.Priority()
		if err != nil {
			panic(fmt.Errorf("invalid priority in group %v: %s", g[j], err))
		}
	}
	return v1 < v2
}

func (g groupsByPriority) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

type groupListByPriority [][]*assets.Group

// each group list get the lowest of the declared priorities
func (g groupListByPriority) groupPriority(i int) int {
	prio := 0
	for _, v := range g[i] {
		if v.Options != nil {
			if val, _ := v.Options.Priority(); val != 0 && (prio == 0 || val < prio) {
				prio = val
			}
		}
	}
	return prio
}

func (g groupListByPriority) Len() int {
	return len(g)
}

func (g groupListByPriority) Less(i, j int) bool {
	return g.groupPriority(i) < g.groupPriority(j)
}

func (g groupListByPriority) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

func sortGroups(groups [][]*assets.Group) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	// Sort each group list by priority
	for _, v := range groups {
		sort.Stable(groupsByPriority(v))
	}
	// Sort the list of all groups by priority
	sort.Stable(groupListByPriority(groups))
	return
}

func (t *Template) prepareAssets() error {
	groups, err := t.preparedAssetsGroups(t.vars, t, nil)
	if err != nil {
		return err
	}
	if err := sortGroups(groups); err != nil {
		return err
	}
	var top bytes.Buffer
	var bottom bytes.Buffer
	for _, group := range groups {
		// Only bundle and use CDNs in non-debug mode
		if !t.Debug {
			if group[0].Options.Bundle() || group[0].Options.Bundable() {
				bundled, err := assets.Bundle(group, group[0].Options)
				if err == nil {
					group = []*assets.Group{
						&assets.Group{
							Manager: group[0].Manager,
							Assets:  []*assets.Asset{bundled},
							Options: group[0].Options,
						},
					}
				} else {
					var names []string
					for _, g := range group {
						for _, a := range g.Assets {
							names = append(names, a.Name)
						}
					}
					log.Errorf("error bundling assets %s: %s - using individual assets", names, err)
				}
			} else if group[0].Options.Cdn() {
				for _, g := range group {
					var groupAssets []*assets.Asset
					for _, a := range g.Assets {
						cdnAssets, err := assets.CdnAssets(g.Manager, a)
						if err != nil {
							if !g.Manager.Has(a.Name) {
								return fmt.Errorf("could not find CDN for asset %q: %s", a.Name, err)
							}
							log.Warningf("could not find CDN for asset %q: %s - using local copy", a.Name, err)
							groupAssets = append(groupAssets, a)
							continue
						}
						groupAssets = append(groupAssets, cdnAssets...)
					}
					g.Assets = groupAssets
				}
			}
		}
		for _, g := range group {
			for _, v := range g.Assets {
				switch v.Position {
				case assets.Top:
					if err := assets.RenderTo(&top, g.Manager, v); err != nil {
						return fmt.Errorf("error rendering asset %q", v.Name)
					}
					top.WriteByte('\n')
				case assets.Bottom:
					if err := assets.RenderTo(&bottom, g.Manager, v); err != nil {
						return fmt.Errorf("error rendering asset %q", v.Name)
					}
					bottom.WriteByte('\n')
				default:
					return fmt.Errorf("asset %q has invalid position %s", v.Name, v.Position)
				}
			}
		}
	}
	t.topAssets = top.Bytes()
	t.bottomAssets = bottom.Bytes()
	return nil
}

func (t *Template) prepareHooks() error {
	for _, v := range t.hooks {
		var key string
		switch v.Position {
		case assets.Top:
			key = topBoilerplateName
		case assets.Bottom:
			key = bottomBoilerplateName
		case assets.None:
			// must be manually referenced from
			// another template
		default:
			return fmt.Errorf("invalid hook position %d", v.Position)
		}
		if key != "" {
			node := &parse.TemplateNode{
				NodeType: parse.NodeTemplate,
				Name:     v.Template.qname(v.Template.root),
				Pipe: &parse.PipeNode{
					NodeType: parse.NodePipe,
					Cmds: []*parse.CommandNode{
						&parse.CommandNode{
							NodeType: parse.NodeCommand,
							Args:     []parse.Node{&parse.DotNode{}},
						},
					},
				},
			}
			tree := t.trees[key].Copy()
			tree.Root.Nodes = append(tree.Root.Nodes, node)
			t.trees[key] = tree
			// we must regenerate the html/template structure, so
			// it sees this new template call and escapes it.
			t.rebuild()
		}
	}
	return nil
}

func (t *Template) evalCommentVar(varname string) (string, error) {
	return eval(t.vars, varname)
}

func (t *Template) parseCommentVariables(values []string) ([]string, error) {
	parsed := make([]string, len(values))
	for ii, v := range values {
		s := strings.Index(v, "{{")
		for s >= 0 {
			end := strings.Index(v[s:], "}}")
			if end < 0 {
				return nil, fmt.Errorf("unterminated variable %q", v[s:])
			}
			// Adjust end to be relative to the start of the string
			end += s
			varname := strings.TrimSpace(v[s+2 : end])
			if len(varname) == 0 {
				return nil, fmt.Errorf("empty variable name")
			}
			if varname[0] != '@' {
				return nil, fmt.Errorf("invalid variable name %q, must start with @", varname)
			}
			varname = varname[1:]
			if len(varname) == 0 {
				return nil, fmt.Errorf("empty variable name")
			}
			value, err := t.evalCommentVar(varname)
			if err != nil {
				return nil, fmt.Errorf("error evaluating variable %q: %s", varname, err)
			}
			v = v[:s] + value + v[end+2:]
			s = strings.Index(v, "{{")
		}
		parsed[ii] = v
	}
	return parsed, nil
}

func (t *Template) parseComment(name string, comment string, file string, included bool) error {
	lines := stringutil.SplitLines(comment)
	extended := false
	for _, v := range lines {
		m := keyRe.FindStringSubmatchIndex(v)
		if m != nil && m[0] == 0 && len(m) > 3 {
			start := m[1] - m[3]
			end := start + m[2]
			key := strings.ToLower(strings.TrimSpace(v[start:end]))
			var options assets.Options
			var value string
			if len(v) > end {
				rem := v[end+1:]
				if v[end] == '|' {
					// Has options
					colon := strings.IndexByte(rem, ':')
					opts := rem[:colon]
					var err error
					options, err = assets.ParseOptions(opts)
					if err != nil {
						return fmt.Errorf("error parsing options for asset key %q: %s", key, err)
					}
					value = rem[colon+1:]
				} else {
					// No options
					value = rem
				}
			}
			splitted, err := stringutil.SplitFields(value, ",")
			if err != nil {
				return fmt.Errorf("error parsing value for asset key %q: %s", key, err)
			}
			values, err := t.parseCommentVariables(splitted)
			if err != nil {
				return fmt.Errorf("error parsing values for asset key %q: %s", key, err)
			}

			inc := true
			switch key {
			case "extend", "extends":
				if extended || len(values) > 1 {
					return fmt.Errorf("templates can only extend one template")
				}
				if t.final {
					return fmt.Errorf("template has been declared as final")
				}
				if strings.ToLower(values[0]) == "none" {
					t.final = true
					break
				}
				extended = true
				inc = false
				fallthrough
			case "include", "includes":
				for _, n := range values {
					err := t.load(n, inc, name)
					if err != nil {
						return err
					}
				}
			case "content-type", "mime-type":
				switch len(values) {
				case 1:
					t.contentType = values[0]
				case 2:
					t.contentType = fmt.Sprintf("%s; charset=%s", values[0], values[1])
				default:
					return fmt.Errorf("%q must have one or two values (e.g. \"text/html\" or \"text/html, UTF-8\" - without quotes", key)
				}
			default:
				if t.AssetsManager == nil {
					return ErrNoAssetsManager
				}
				group, err := assets.Parse(t.AssetsManager, key, values, options)
				if err != nil {
					return err
				}
				t.assetGroups = append(t.assetGroups, group)
			}
		}
	}
	if !extended && !included {
		t.root = file
	}
	return nil
}

func (t *Template) loadText(name string) (string, error) {
	f, _, err := t.loader.Load(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	if conv := converters[strings.ToLower(path.Ext(name))]; conv != nil {
		b, err = conv(b)
		if err != nil {
			return "", err
		}
	}
	s := string(b)
	return s, nil
}

func (t *Template) load(name string, included bool, from string) error {
	for _, v := range t.loaded {
		if v == name {
			return nil
		}
	}
	s, err := t.loadText(name)
	if err != nil {
		return err
	}
	matches := commentRe.FindStringSubmatch(s)
	comment := ""
	if matches != nil && len(matches) > 0 {
		comment = matches[1]
	}
	// Add it to the loaded list before calling parseComment, since
	// it's the path which can trigger additional loads.
	t.loaded = append(t.loaded, name)
	err = t.parseComment(name, comment, name, included)
	if err != nil {
		return err
	}
	if idx := strings.Index(s, "</head>"); idx >= 0 {
		s = s[:idx] + fmt.Sprintf("{{ template %q . }}", topBoilerplateName) + s[idx:]
	}
	if idx := strings.Index(s, "</body>"); idx >= 0 {
		s = s[:idx] + fmt.Sprintf("{{ template %q . }}", bottomBoilerplateName) + s[idx:]
	}
	// Replace {{ block }} with {{ template }} + {{ define }}
	s, err = replaceBlocks(name, s)
	if err != nil {
		return err
	}
	// The $Vars definition must be present at parse
	// time, because otherwise the parser will throw an
	// error when it finds a variable which wasn't
	// previously defined
	// Prepend to the template and to any define nodes found
	s = templatePrepend + defineRe.ReplaceAllString(s, "$0"+strings.Replace(templatePrepend, "$", "$$", -1))
	// Replace @variable shorthands
	s = templateutil.ReplaceVariableShorthands(s, '@', varsKey)
	var fns map[string]interface{}
	if t.funcMap != nil {
		fns = t.funcMap.asTemplateFuncs()
	}
	treeMap, err := parse.Parse(name, s, leftDelim, rightDelim, templateFuncs.asTemplateFuncs(), fns)
	if err != nil {
		return err
	}
	if err := t.replaceExtendTag(name, treeMap, from); err != nil {
		return err
	}
	var renames map[string]string
	for k, v := range treeMap {
		// Remove the first node, since it's there to set
		// $Vars := .Vars, so the parser does not complain
		v.Root.Nodes = v.Root.Nodes[1:]
		// Remove the \n introduced by the prepended text
		for _, n := range v.Root.Nodes {
			if tn, ok := n.(*parse.TextNode); ok {
				tn.Text = tn.Text[1:]
				break
			}
		}
		if _, contains := t.trees[k]; contains {
			log.Debugf("Template %s redefined", k)
			// Redefinition of a template, which is allowed
			// by gondola templates. Just rename this
			// template and change any template
			// nodes referring to it in the final sweep
			if renames == nil {
				renames = make(map[string]string)
			}
			fk := k
			for {
				k += "_"
				if len(renames[fk]) < len(k) {
					renames[fk] = k
					break
				}
			}
		}
		t.texts[k] = s
		err := t.AddParseTree(k, v)
		if err != nil {
			return err
		}
	}
	if t.contentType == "" {
		mimeType := mime.TypeByExtension(path.Ext(t.name))
		if mimeType == "" {
			mimeType = "text/html; charset=utf-8"
		}
		t.contentType = mimeType
	}
	if renames != nil {
		t.renameTemplates(renames)
	}
	return nil
}

func (t *Template) replaceExtendTag(name string, treeMap map[string]*parse.Tree, from string) error {
	var err error
	hasExtend := false
	var loc string
	for _, v := range treeMap {
		templateutil.WalkTree(v, func(n, p parse.Node) {
			if err != nil {
				return
			}
			if templateutil.IsPseudoFunction(n, "extend") {
				if hasExtend {
					loc2, _ := v.ErrorContext(n)
					err = fmt.Errorf("multiple {{ extend }} tags in %q, %s and %s", name, loc, loc2)
					return
				}
				hasExtend = true
				loc, _ = v.ErrorContext(n)
				var repl parse.Node
				if from == "" {
					// empty
					log.Debugf("removing {{ extend }} from %q", name)
					repl = &parse.TextNode{
						NodeType: parse.NodeText,
						Pos:      n.Position(),
					}
				} else {
					log.Debugf("extending %q at %s with %q", name, loc, from)
					repl = templateutil.TemplateNode(from, n.Position())
				}
				err = templateutil.ReplaceNode(n, p, repl)
			}
		})
	}
	return err
}

func (t *Template) walkTrees(nt parse.NodeType, f func(parse.Node)) {
	for _, v := range t.trees {
		templateutil.WalkTree(v, func(n, p parse.Node) {
			if n.Type() == nt {
				f(n)
			}
		})
	}
}

func (t *Template) referencedTemplates() []string {
	var templates []string
	t.walkTrees(parse.NodeTemplate, func(n parse.Node) {
		templates = append(templates, n.(*parse.TemplateNode).Name)
	})
	return templates
}

func (t *Template) assetFunc(arg string) (string, error) {
	if t.AssetsManager != nil {
		return t.AssetsManager.URL(arg), nil
	}
	return "", ErrNoAssetsManager
}

func (t *Template) renameTemplates(renames map[string]string) {
	t.walkTrees(parse.NodeTemplate, func(n parse.Node) {
		node := n.(*parse.TemplateNode)
		if rename, ok := renames[node.Name]; ok {
			node.Name = rename
		}
	})
}

func (t *Template) addHtmlHooks() {
	// Unfortunately, there's a bug in text/template executor which
	// ends up panic'ing if there's an error in a mangled template, so
	// we must wrap this in a recover.
	defer func() {
		recover()
	}()
	// Add the root template when the name "", so html/template
	// can find it and escape all the trees from it
	if _, ok := t.trees[""]; !ok && t.root != "" {
		t.tmpl.AddParseTree("", t.trees[t.root])
	}
	// Need to execute it once, for html/template to add
	// the escaping hooks.
	t.tmpl.Execute(ioutil.Discard, nil)
}

func (t *Template) addHtmlEscaping() {
	t.addHtmlHooks()
	// Add escaping functions
	t.Funcs(htmlEscapeFuncs)
	// html/template might introduce new trees. it renames the
	// template invocations, but we must add the trees ourselves
	for _, v := range t.tmpl.Templates() {
		if v.Tree != nil {
			if t.trees[v.Tree.Name] == nil {
				t.trees[v.Tree.Name] = v.Tree
			}
		}
	}
}

func (t *Template) Funcs(funcs FuncMap) *Template {
	if t.funcMap == nil {
		t.funcMap = make(FuncMap)
	}
	for k, v := range funcs {
		t.funcMap.add(k, v)
	}
	t.tmpl.Funcs(funcs.asTemplateFuncs())
	return t
}

func (t *Template) Include(name string) error {
	err := t.load(name, true, "")
	if err != nil {
		return err
	}
	return nil
}

// Parse parses the template starting with the given
// template name (and following any extends/includes
// directives declared in it).
func (t *Template) Parse(name string) error {
	return t.ParseVars(name, nil)
}

func (t *Template) ParseVars(name string, vars VarMap) error {
	if t.name == "" {
		t.name = name
	}
	t.vars = vars
	err := t.load(name, false, "")
	if err != nil {
		return err
	}
	// Add assets templates
	err = t.AddParseTree(topBoilerplateName, topTree)
	if err != nil {
		return err
	}
	err = t.AddParseTree(bottomBoilerplateName, bottomTree)
	if err != nil {
		return err
	}
	return nil
}

func (t *Template) AddParseTree(name string, tree *parse.Tree) error {
	_, err := t.tmpl.AddParseTree(name, tree)
	if err != nil {
		return err
	}
	t.trees[name] = tree
	return nil
}

func (t *Template) Execute(w io.Writer, data interface{}) error {
	return t.ExecuteTemplateVars(w, t.root, data, nil)
}

func (t *Template) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	return t.ExecuteTemplateVars(w, name, data, nil)
}

func (t *Template) ExecuteVars(w io.Writer, data interface{}, vars VarMap) error {
	return t.ExecuteTemplateVars(w, t.root, data, vars)
}

func (t *Template) ExecuteTemplateVars(w io.Writer, name string, data interface{}, vars VarMap) error {
	if profile.On {
		profile.Startf("template-exec", "%s => %s", t.qname(t.name), name).AutoEnd()
	}
	buf := getBuffer()
	err := t.prog.execute(buf, name, data, vars)
	if err != nil {
		return err
	}
	if t.Minify {
		// Instead of using a new Buffer, make a copy of the []byte and Reset
		// buf. This minimizes the number of allocations while momentarily
		// using a bit more of memory than we need (exactly one byte per space
		// removed in the output).
		b := buf.Bytes()
		bc := make([]byte, len(b))
		copy(bc, b)
		r := bytes.NewReader(bc)
		buf.Reset()
		if err := html.Minify(buf, r); err != nil {
			return err
		}
	}
	if rw, ok := w.(http.ResponseWriter); ok {
		header := rw.Header()
		header.Set("Content-Type", t.contentType)
		header.Set("Content-Length", strconv.Itoa(buf.Len()))
	}
	_, err = w.Write(buf.Bytes())
	putBuffer(buf)
	return err
}

// MustExecute works like Execute, but panics if there's an error
func (t *Template) MustExecute(w io.Writer, data interface{}) {
	err := t.Execute(w, data)
	if err != nil {
		log.Fatalf("Error executing template: %s\n", err)
	}
}

// AddFuncs registers new functions which will be available to
// the templates. Please, note that you must register the functions
// before compiling a template that uses them, otherwise the template
// parser will return an error.
func AddFuncs(f FuncMap) {
	for k, v := range f {
		templateFuncs[k] = v
	}
}

// Returns a loader which loads templates from
// the tmpl directory, relative to the application
// binary.
func DefaultTemplateLoader() loaders.Loader {
	return loaders.FSLoader(pathutil.Relative("tmpl"))
}

// New returns a new template with the given loader and assets
// manager. Please, refer to the documention in gnd.la/loaders
// and gnd.la/asssets for further information in those types.
// If the loader is nil, DefaultTemplateLoader() will be used.
func New(loader loaders.Loader, manager *assets.Manager) *Template {
	if loader == nil {
		loader = DefaultTemplateLoader()
	}
	t := &Template{
		AssetsManager: manager,
		loader:        loader,
		trees:         make(map[string]*parse.Tree),
		texts:         make(map[string]string),
	}
	t.init()
	return t
}

// Parse creates a new template using the given loader and manager and then
// parses the template with the given name.
func Parse(loader loaders.Loader, manager *assets.Manager, name string) (*Template, error) {
	t := New(loader, manager)
	err := t.Parse(name)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// MustParse works like parse, but panics if there's an error
func MustParse(loader loaders.Loader, manager *assets.Manager, name string) *Template {
	t, err := Parse(loader, manager, name)
	if err != nil {
		log.Fatalf("Error loading template %s: %s\n", name, err)
	}
	return t
}

func compileTree(text string) *parse.Tree {
	funcs := template.FuncMap{
		topAssetsFuncName:    nop,
		bottomAssetsFuncName: nop,
	}
	treeMap, err := parse.Parse("", text, leftDelim, rightDelim, funcs)
	if err != nil {
		panic(err)
	}
	return treeMap[""]
}

func canBundle(g1, g2 *assets.Group) bool {
	if g1.Manager == g2.Manager {
		if len(g1.Assets) > 0 && len(g2.Assets) > 0 {
			f1 := g1.Assets[0]
			f2 := g2.Assets[0]
			return f1.Type == f2.Type && f1.Position == f2.Position
		}
	}
	return false
}

func copyGroup(src *assets.Group) *assets.Group {
	copies := make([]*assets.Asset, len(src.Assets))
	for ii, v := range src.Assets {
		a := *v
		copies[ii] = &a
	}
	g := *src
	g.Assets = copies
	return &g
}

func namespacedTree(tree *parse.Tree, ns []string) *parse.Tree {
	tree = tree.Copy()
	if len(ns) > 0 {
		prefix := strings.Join(ns, nsSep) + nsMark
		templateutil.WalkTree(tree, func(n, p parse.Node) {
			if n.Type() == parse.NodeTemplate {
				tmpl := n.(*parse.TemplateNode)
				tmpl.Name = prefix + tmpl.Name
			}
		})
	}
	return tree
}

func errorContext(name string, s string, pos int) string {
	var text string
	if pos <= len(s) {
		text = s[:pos]
	}
	var col int
	idx := strings.LastIndex(text, "\n")
	if idx == -1 {
		col = pos
	} else {
		col = pos - idx - 1
	}
	line := 1 + strings.Count(text, "\n")
	return fmt.Sprintf("%s:%d:%d", name, line, col)
}

func replaceBlocks(name string, s string) (string, error) {
	for m := blockRe.FindStringSubmatchIndex(s); m != nil; m = blockRe.FindStringSubmatchIndex(s) {
		all := s[m[0]:m[1]]
		name := s[m[2]:m[3]]
		if name == "" {
			return "", fmt.Errorf("%s: invalid {{ block }} tag %q, missing name (must be quoted)", errorContext(name, s, m[0]), all)
		}
		if _, err := strconv.Unquote(name); err != nil {
			return "", fmt.Errorf("%s: invalid {{ block }} tag %q, name is not correctly quoted: %s", errorContext(name, s, m[0]), all, err)
		}
		rem := s[m[3]:m[4]]
		if rem != "" {
			return "", fmt.Errorf("%s: invalid {{ block }} tag %q, extra content after template name %q", errorContext(name, s, m[0]), all, rem)
		}
		s = fmt.Sprintf("%s{{ template %s . }}{{ define %s }}%s", s[:m[0]], name, name, s[m[1]:])
	}
	return s, nil
}

func NamespacedName(ns []string, name string) string {
	if len(ns) > 0 {
		return strings.Join(ns, nsSep) + nsMark + name
	}
	return name
}

func nop() interface{} { return nil }
