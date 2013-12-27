package template

import (
	"bytes"
	"errors"
	"fmt"
	"gnd.la/html"
	"gnd.la/loaders"
	"gnd.la/log"
	"gnd.la/template/assets"
	"gnd.la/util"
	"gnd.la/util/internal/templateutil"
	"gnd.la/util/textutil"
	"html/template"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template/parse"
)

type FuncMap map[string]interface{}
type VarMap map[string]interface{}

const (
	leftDelim               = "{{"
	rightDelim              = "}}"
	TopAssetsTmplName       = "TopAssets"
	BottomAssetsTmplName    = "BottomAssets"
	dataKey                 = "Data"
	topAssetsBoilerplate    = "{{ range __topAssets }}{{ render . }}\n{{ end }}"
	bottomAssetsBoilerplate = "{{ range __bottomAssets }}{{ render . }}\n{{ end }}"
)

var (
	ErrNoAssetsManager = errors.New("Template does not have an assets manager")
	commentRe          = regexp.MustCompile(`(?s:\{\{\\*(.*?)\*/\}\})`)
	keyRe              = regexp.MustCompile(`(?s:\s*([\w\-_])+?(:|\|))`)
	defineRe           = regexp.MustCompile(`(\{\{\s*?define.*?\}\})`)
	topTree            = compileTree(TopAssetsTmplName, topAssetsBoilerplate)
	bottomTree         = compileTree(BottomAssetsTmplName, bottomAssetsBoilerplate)
)

type Template struct {
	*template.Template
	Name          string
	Debug         bool
	Loader        loaders.Loader
	AssetsManager assets.Manager
	Trees         map[string]*parse.Tree
	Minify        bool
	funcMap       FuncMap
	root          string
	topAssets     []assets.Asset
	bottomAssets  []assets.Asset
	vars          VarMap
	prepend       string
	renames       map[string]string
	contentType   string
}

func (t *Template) Root() string {
	return t.root
}

func (t *Template) Assets() []assets.Asset {
	var ass []assets.Asset
	ass = append(ass, t.topAssets...)
	ass = append(ass, t.bottomAssets...)
	return ass
}

func (t *Template) AddAssets(ass []assets.Asset) {
	for _, v := range ass {
		if v.AssetPosition() == assets.Top {
			t.topAssets = append(t.topAssets, v)
		} else {
			t.bottomAssets = append(t.bottomAssets, v)
		}
	}
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
			if varname[0] != '$' {
				return nil, fmt.Errorf("invalid variable name %q, must start with $", varname)
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

func (t *Template) parseComment(comment string, file string, included bool) error {
	// Escaped newlines
	comment = strings.Replace(comment, "\\\n", " ", -1)
	lines := strings.Split(comment, "\n")
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
			splitted, err := textutil.SplitFields(value, ",")
			if err != nil {
				return fmt.Errorf("error parsing value for asset key %q: %s", key, err)
			}
			values, err := t.parseCommentVariables(splitted)
			if err != nil {
				return fmt.Errorf("error parsing values for asset key %q: %s", key, err)
			}
			for ii, val := range values {
				// Check if the asset is a template
				if val != "" && val[0] == '(' && val[len(val)-1] == ')' {
					name := val[1 : len(val)-1]
					var err error
					values[ii], err = executeAsset(t, name)
					if err != nil {
						return fmt.Errorf("error executing asset template %s: %s", name, err)
					}
				}
			}

			inc := true
			switch key {
			case "extend", "extends":
				if extended || len(values) > 1 {
					return fmt.Errorf("templates can only extend one template")
				}
				extended = true
				inc = false
				fallthrough
			case "include", "includes":
				for _, n := range values {
					err := t.load(n, inc)
					if err != nil {
						return err
					}
				}
			default:
				if t.AssetsManager == nil {
					return ErrNoAssetsManager
				}
				ass, err := assets.Parse(t.AssetsManager, key, values, options)
				if err != nil {
					return err
				}
				for _, a := range ass {
					if a.AssetPosition() == assets.Top {
						t.topAssets = append(t.topAssets, a)
					} else {
						t.bottomAssets = append(t.bottomAssets, a)
					}
				}
			}
		}
	}
	if !extended && !included {
		t.root = file
	}
	return nil
}

func (t *Template) load(name string, included bool) error {
	// TODO: Detect circular dependencies
	f, _, err := t.Loader.Load(name)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	if conv := converters[strings.ToLower(path.Ext(name))]; conv != nil {
		b, err = conv(b)
		if err != nil {
			return err
		}
	}
	s := string(b)
	matches := commentRe.FindStringSubmatch(s)
	comment := ""
	if matches != nil && len(matches) > 0 {
		comment = matches[1]
	}
	err = t.parseComment(comment, name, included)
	if err != nil {
		return err
	}
	if idx := strings.Index(s, "</head>"); idx >= 0 {
		s = s[:idx] + fmt.Sprintf("{{ template \"%s\" }}", TopAssetsTmplName) + s[idx:]
	}
	if idx := strings.Index(s, "</body>"); idx >= 0 {
		s = s[:idx] + fmt.Sprintf("{{ template \"%s\" }}", BottomAssetsTmplName) + s[idx:]
	}
	if t.prepend != "" {
		// Prepend to the template and to any define nodes found
		s = t.prepend + defineRe.ReplaceAllString(s, "$0"+strings.Replace(t.prepend, "$", "$$", -1))
	}
	treeMap, err := parse.Parse(name, s, leftDelim, rightDelim, templateFuncs, t.funcMap)
	if err != nil {
		return err
	}
	for k, v := range treeMap {
		if _, contains := t.Trees[k]; contains {
			log.Debugf("Template %s redefined", k)
			// Redefinition of a template, which is allowed
			// by gondola templates. Just rename this
			// template and change any template
			// nodes referring to it in the final sweep
			if t.renames == nil {
				t.renames = make(map[string]string)
			}
			fk := k
			for {
				k += "_"
				if len(t.renames[fk]) < len(k) {
					t.renames[fk] = k
					break
				}
			}
		}
		err := t.AddParseTree(k, v)
		if err != nil {
			return err
		}
	}
	mimeType := mime.TypeByExtension(path.Ext(t.Name))
	if mimeType == "" {
		mimeType = "text/html; charset=utf-8"
	}
	t.contentType = mimeType
	return nil
}

func (t *Template) walkTrees(nt parse.NodeType, f func(parse.Node)) {
	for _, v := range t.Trees {
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

func (t *Template) Funcs(funcs FuncMap) {
	if t.funcMap == nil {
		t.funcMap = make(FuncMap)
	}
	for k, v := range funcs {
		t.funcMap[k] = v
	}
	t.Template.Funcs(template.FuncMap(t.funcMap))
}

func (t *Template) Include(name string) error {
	err := t.load(name, true)
	if err != nil {
		return err
	}
	t.walkTrees(parse.NodeTemplate, func(n parse.Node) {
		node := n.(*parse.TemplateNode)
		if rename, ok := t.renames[node.Name]; ok {
			node.Name = rename
		}
	})
	return nil
}

// Parse parses the template starting with the given
// template name (and following any extends/includes
// directives declared in it).
func (t *Template) Parse(name string) error {
	return t.ParseVars(name, nil)
}

// ParseVars works like Parse, but it also inserts predefined
// variables in the template. The values in vars will be the
// defaults and may be overriden by using ExecuteVars.
func (t *Template) ParseVars(name string, vars VarMap) error {
	t.Name = name
	if len(vars) > 0 {
		t.vars = vars
		// The variable definitions must be present at parse
		// time, because otherwise the parser will throw an
		// error when it finds a variable which wasn't
		// previously defined
		var p []string
		for k, _ := range vars {
			p = append(p, fmt.Sprintf("{{ $%s := .%s }}", k, k))
		}
		t.prepend = strings.Join(p, "")
	}
	err := t.load(name, false)
	if err != nil {
		return err
	}
	/* Add assets */
	err = t.AddParseTree(TopAssetsTmplName, topTree)
	if err != nil {
		return err
	}
	err = t.AddParseTree(BottomAssetsTmplName, bottomTree)
	if err != nil {
		return err
	}
	// Don't define missing templates to be empty, since
	// it causes more problems than it solves.
	//
	// Fill any empty templates, so we allow templates
	// to be left undefined
	//for _, v := range t.referencedTemplates() {
	//	if _, ok := t.Trees[v]; !ok {
	//		tree := compileTree(v, "")
	//		t.AddParseTree(v, tree)
	//	}
	//}
	var templateArgs []parse.Node
	if n := len(vars); n > 0 {
		// Modify the parse trees to always define vars
		for _, tr := range t.Trees {
			if len(tr.Root.Nodes) < n {
				/* Empty template */
				continue
			}
			// Skip the first n nodes, since they set the variables.
			// Then wrap the rest of template in a WithNode, which sets
			// the dot to .Data
			field := &parse.FieldNode{
				NodeType: parse.NodeField,
				Ident:    []string{dataKey},
			}
			command := &parse.CommandNode{
				NodeType: parse.NodeCommand,
				Args:     []parse.Node{field},
			}
			pipe := &parse.PipeNode{
				NodeType: parse.NodePipe,
				Cmds:     []*parse.CommandNode{command},
			}
			var nodes []parse.Node
			nodes = append(nodes, tr.Root.Nodes[:n]...)
			root := tr.Root.Nodes[n:]
			newRoot := &parse.ListNode{
				NodeType: parse.NodeList,
				Nodes:    root,
			}
			// The list needs to be copied, otherwise the
			// html/template escaper complains that the
			// node is shared between templates
			with := &parse.WithNode{
				parse.BranchNode{
					NodeType: parse.NodeWith,
					Pipe:     pipe,
					List:     newRoot,
					ElseList: newRoot.CopyList(),
				},
			}
			nodes = append(nodes, with)
			tr.Root = &parse.ListNode{
				NodeType: parse.NodeList,
				Nodes:    nodes,
			}
		}
		// Rewrite any template nodes to pass also the variables, since
		// they are not inherited
		templateArgs = []parse.Node{parse.NewIdentifier("map")}
		for k, _ := range vars {
			templateArgs = append(templateArgs, &parse.StringNode{
				NodeType: parse.NodeString,
				Quoted:   fmt.Sprintf("\"%s\"", k),
				Text:     k,
			})
			templateArgs = append(templateArgs, &parse.VariableNode{
				NodeType: parse.NodeVariable,
				Ident:    []string{fmt.Sprintf("$%s", k)},
			})
		}
		templateArgs = append(templateArgs, &parse.StringNode{
			NodeType: parse.NodeString,
			Quoted:   fmt.Sprintf("\"%s\"", dataKey),
			Text:     dataKey,
		})
	}

	if len(t.renames) > 0 || len(templateArgs) > 0 {
		t.walkTrees(parse.NodeTemplate, func(n parse.Node) {
			node := n.(*parse.TemplateNode)
			if rename, ok := t.renames[node.Name]; ok {
				node.Name = rename
			}
			if templateArgs != nil {
				if node.Pipe == nil {
					// No data, just pass variables
					command := &parse.CommandNode{
						NodeType: parse.NodeCommand,
						Args:     templateArgs[:len(templateArgs)-1],
					}
					node.Pipe = &parse.PipeNode{
						NodeType: parse.NodePipe,
						Cmds:     []*parse.CommandNode{command},
					}
				} else {
					newPipe := &parse.PipeNode{
						NodeType: parse.NodePipe,
						Cmds:     node.Pipe.Cmds,
					}
					args := make([]parse.Node, len(templateArgs))
					copy(args, templateArgs)
					command := &parse.CommandNode{
						NodeType: parse.NodeCommand,
						Args:     append(args, newPipe),
					}
					node.Pipe.Cmds = []*parse.CommandNode{command}
				}
			}
		})
	}
	return nil
}

func (t *Template) AddParseTree(name string, tree *parse.Tree) error {
	_, err := t.Template.AddParseTree(name, tree)
	if err != nil {
		return err
	}
	t.Trees[name] = tree
	return nil
}

func (t *Template) Execute(w io.Writer, data interface{}) error {
	return t.ExecuteTemplateVars(w, "", data, nil)
}

func (t *Template) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	return t.ExecuteTemplateVars(w, name, data, nil)
}

func (t *Template) ExecuteVars(w io.Writer, data interface{}, vars VarMap) error {
	return t.ExecuteTemplateVars(w, "", data, vars)
}

func (t *Template) ExecuteTemplateVars(w io.Writer, name string, data interface{}, vars VarMap) error {
	// TODO: Make sure vars is the same as the vars that were compiled in
	var templateData interface{}
	if len(vars) > 0 {
		combined := make(map[string]interface{}, len(t.vars))
		for k, v := range t.vars {
			if iv, ok := vars[k]; ok {
				combined[k] = iv
			} else {
				combined[k] = v
			}
		}
		combined[dataKey] = data
		templateData = combined
	} else {
		templateData = data
	}
	var buf bytes.Buffer
	if name == "" {
		name = t.root
	}
	err := t.Template.ExecuteTemplate(&buf, name, templateData)
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
		if err := html.Minify(&buf, r); err != nil {
			return err
		}
	}
	if rw, ok := w.(http.ResponseWriter); ok {
		header := rw.Header()
		header.Set("Content-Type", t.contentType)
		header.Set("Content-Length", strconv.Itoa(buf.Len()))
	}
	_, err = w.Write(buf.Bytes())
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
	return loaders.FSLoader(util.RelativePath("tmpl"))
}

// New returns a new template with the given loader and assets
// manager. Please, refer to the documention in gnd.la/loaders
// and gnd.la/asssets for further information in those types.
// If the loader is nil, DefaultTemplateLoader() will be used.
func New(loader loaders.Loader, manager assets.Manager) *Template {
	if loader == nil {
		loader = DefaultTemplateLoader()
	}
	t := &Template{
		Template:      template.New(""),
		Loader:        loader,
		AssetsManager: manager,
		Trees:         make(map[string]*parse.Tree),
	}
	// This is required so text/template calls t.init()
	// and initializes the common data structure
	t.Template.New("")
	funcs := FuncMap{
		"__topAssets":    func() []assets.Asset { return t.topAssets },
		"__bottomAssets": func() []assets.Asset { return t.bottomAssets },
		"asset": func(arg string) (string, error) {
			if t.AssetsManager != nil {
				return t.AssetsManager.URL(arg), nil
			}
			return "", ErrNoAssetsManager
		},
	}
	t.Funcs(funcs)
	t.Template.Funcs(template.FuncMap(funcs)).Funcs(templateFuncs)
	return t
}

// Parse creates a new template using the given loader and manager and then
// parses the template with the given name.
func Parse(loader loaders.Loader, manager assets.Manager, name string) (*Template, error) {
	t := New(loader, manager)
	err := t.Parse(name)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// MustParse works like parse, but panics if there's an error
func MustParse(loader loaders.Loader, manager assets.Manager, name string) *Template {
	t, err := Parse(loader, manager, name)
	if err != nil {
		log.Fatalf("Error loading template %s: %s\n", name, err)
	}
	return t
}

func compileTree(name, text string) *parse.Tree {
	funcs := map[string]interface{}{
		"__topAssets":    func() {},
		"__bottomAssets": func() {},
		"asset":          func() {},
		"render":         func() {},
	}
	treeMap, err := parse.Parse(name, text, leftDelim, rightDelim, funcs)
	if err != nil {
		panic(err)
	}
	return treeMap[name]
}
