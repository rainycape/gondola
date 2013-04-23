package template

import (
	"bytes"
	"fmt"
	"gondola/assets"
	"gondola/files"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template/parse"
)

type FuncMap map[string]interface{}

const (
	leftDelim               = "{{"
	rightDelim              = "}}"
	TopAssetsTmplName       = "TopAssets"
	BottomAssetsTmplName    = "BottomAssets"
	dataKey                 = "Data"
	topAssetsBoilerplate    = "{{ range __topAssets }}\n{{ .HTML }}{{ end }}"
	bottomAssetsBoilerplate = "{{ range __bottomAssets }}\n{{ .HTML }}{{ end }}"
)

var (
	commentRe  = regexp.MustCompile(`(?s:\{\{\\*(.*?)\*/\}\})`)
	keyRe      = regexp.MustCompile(`(?s:\s*([\w\-_])+?(:|/))`)
	defineRe   = regexp.MustCompile(`(\{\{\s*?define.*?\}\})`)
	topTree    = compileTree(TopAssetsTmplName, topAssetsBoilerplate)
	bottomTree = compileTree(BottomAssetsTmplName, bottomAssetsBoilerplate)
)

type Template struct {
	*template.Template
	Dir          string
	AssetsPrefix string
	Trees        map[string]*parse.Tree
	funcMap      FuncMap
	root         string
	topAssets    []assets.Asset
	bottomAssets []assets.Asset
	vars         []string
	renames      map[string]string
}

func (t *Template) readString(idx *int, s string, stopchars string) (string, error) {
	var value string
	start := *idx
	if s[*idx] == '"' {
		(*idx)++
		for ; *idx < len(s); (*idx)++ {
			if s[*idx] == '"' && s[((*idx)-1)] != '\\' {
				value = s[start+1 : *idx]
				break
			}
		}
	} else {
		for ; *idx < len(s); (*idx)++ {
			if strings.Contains(stopchars, string(s[*idx])) {
				value = s[start:*idx]
				(*idx)--
				break
			}
		}
	}
	if value == "" {
		value = s[start:]
	}
	return value, nil
}

func (t *Template) parseOptions(idx int, line string, remainder string) (options map[string]string, value string, err error) {
	options = make(map[string]string)
	if len(remainder) == 0 || (remainder[0] != ':' && remainder[0] != '/') {
		err = fmt.Errorf("Malformed asset line %d %q", idx, line)
		return
	}
	var key string
	for ii := 0; ii < len(remainder); ii++ {
		ch := remainder[ii]
		if ch == ':' {
			value = strings.TrimSpace(remainder[ii+1:])
			break
		} else if ch == '/' || ch == ',' {
			continue
		} else {
			if key == "" {
				key, err = t.readString(&ii, remainder, "=,:")
				if err != nil {
					return
				}
			} else {
				var val string
				if ch == '=' {
					ii++
					if ii < len(remainder) {
						val, err = t.readString(&ii, remainder, ",:")
						if err != nil {
							return
						}
					}
				}
				options[key] = val
				key = ""
			}
		}
	}
	return
}

func (t *Template) parseComment(comment string, file string, prepend string, included bool) error {
	// Escaped newlines
	comment = strings.Replace(comment, "\\\n", " ", -1)
	lines := strings.Split(comment, "\n")
	extended := false
	for ii, v := range lines {
		m := keyRe.FindStringSubmatchIndex(v)
		if m != nil && m[0] == 0 && len(m) > 3 {
			start := m[1] - m[3]
			end := start + m[2]
			key := strings.ToLower(strings.TrimSpace(v[start:end]))
			options, value, err := t.parseOptions(ii, v, v[m[2]+1:])
			if err != nil {
				return err
			}
			inc := true
			if value != "" {
				switch key {
				case "extend", "extends":
					extended = true
					inc = false
					fallthrough
				case "include", "includes":
					err := t.load(value, prepend, inc)
					if err != nil {
						return err
					}
				default:
					var names []string
					for _, n := range strings.Split(value, ",") {
						names = append(names, strings.TrimSpace(n))
					}
					ass, err := assets.Parse(t.AssetsPrefix, key, options, names)
					if err != nil {
						return err
					}
					for _, a := range ass {
						if a.Position() == assets.Top {
							t.topAssets = append(t.topAssets, a)
						} else {
							t.bottomAssets = append(t.bottomAssets, a)
						}
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

func (t *Template) load(file string, prepend string, included bool) error {
	// TODO: Detect circular dependencies
	b, err := ioutil.ReadFile(path.Join(t.Dir, file))
	if err != nil {
		return err
	}
	s := string(b)
	matches := commentRe.FindStringSubmatch(s)
	comment := ""
	if matches != nil && len(matches) > 0 {
		comment = matches[1]
	}
	err = t.parseComment(comment, file, prepend, included)
	if err != nil {
		return err
	}
	if idx := strings.Index(s, "</head>"); idx >= 0 {
		s = s[:idx] + fmt.Sprintf("{{ template \"%s\" }}", TopAssetsTmplName) + s[idx:]
	}
	if idx := strings.Index(s, "</body>"); idx >= 0 {
		s = s[:idx] + fmt.Sprintf("{{ template \"%s\" }}", BottomAssetsTmplName) + s[idx:]
	}
	if prepend != "" {
		// Prepend to the template and to any define nodes found
		s = prepend + defineRe.ReplaceAllString(s, "$0"+strings.Replace(prepend, "$", "$$", -1))
	}
	treeMap, err := parse.Parse(file, s, leftDelim, rightDelim, templateFuncs, t.funcMap)
	if err != nil {
		return err
	}
	for k, v := range treeMap {
		if _, contains := t.Trees[k]; contains {
			// Redefinition of a template, which is allowed
			// by gondola templates. Just rename this
			// template and change update any template
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
	return nil
}

func (t *Template) walkNode(node parse.Node, nt parse.NodeType, f func(parse.Node)) {
	if node == nil {
		return
	}
	if node.Type() == nt {
		f(node)
	}
	switch x := node.(type) {
	case *parse.ListNode:
		for _, v := range x.Nodes {
			t.walkNode(v, nt, f)
		}
	case *parse.IfNode:
		if x.List != nil {
			t.walkNode(x.List, nt, f)
		}
		if x.ElseList != nil {
			t.walkNode(x.ElseList, nt, f)
		}
	case *parse.WithNode:
		if x.List != nil {
			t.walkNode(x.List, nt, f)
		}
		if x.ElseList != nil {
			t.walkNode(x.ElseList, nt, f)
		}
	case *parse.RangeNode:
		if x.List != nil {
			t.walkNode(x.List, nt, f)
		}
		if x.ElseList != nil {
			t.walkNode(x.ElseList, nt, f)
		}
	}
}

func (t *Template) walkTrees(nt parse.NodeType, f func(parse.Node)) {
	for _, v := range t.Trees {
		t.walkNode(v.Root, nt, f)
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

func (t *Template) Parse(file string) error {
	return t.ParseVars(file, nil)
}

func (t *Template) ParseVars(file string, vars []string) error {
	prepend := ""
	if len(vars) > 0 {
		t.vars = vars
		// The variable definitions must be present at parse
		// time, because otherwise the parser will throw an
		// error when it finds a variable which wasn't
		// previously defined
		var p []string
		for _, v := range vars {
			p = append(p, fmt.Sprintf("{{ $%s := .%s }}", v, v))
		}
		prepend = strings.Join(p, "")
	}
	err := t.load(file, prepend, false)
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
	// Fill any empty templates, so we allow templates
	// to be left undefined
	for _, v := range t.referencedTemplates() {
		if _, ok := t.Trees[v]; !ok {
			tree := compileTree(v, "")
			t.AddParseTree(v, tree)
		}
	}
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
		for _, v := range vars {
			templateArgs = append(templateArgs, &parse.StringNode{
				NodeType: parse.NodeString,
				Quoted:   fmt.Sprintf("\"%s\"", v),
				Text:     v,
			})
			templateArgs = append(templateArgs, &parse.VariableNode{
				NodeType: parse.NodeVariable,
				Ident:    []string{fmt.Sprintf("$%s", v)},
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
	return t.ExecuteVars(w, data, nil)
}

func (t *Template) ExecuteVars(w io.Writer, data interface{}, vars map[string]interface{}) error {
	// TODO: Make sure vars is the same as the vars that were compiled in
	var buf bytes.Buffer
	var templateData interface{}
	if len(vars) > 0 {
		combined := make(map[string]interface{})
		for k, v := range vars {
			combined[k] = v
		}
		combined[dataKey] = data
		templateData = combined
	} else {
		templateData = data
	}
	err := t.ExecuteTemplate(&buf, t.root, templateData)
	if err != nil {
		return err
	}
	if rw, ok := w.(http.ResponseWriter); ok {
		header := rw.Header()
		header.Set("Content-Type", "text/html; charset=utf-8")
		header.Set("Content-Length", strconv.Itoa(buf.Len()))
		rw.Write(buf.Bytes())
	}
	return nil
}

func (t *Template) MustExecute(w io.Writer, data interface{}) {
	err := t.Execute(w, data)
	if err != nil {
		log.Panicf("Error executing template: %s\n", err)
	}
}

func AddFunc(name string, f interface{}) {
	templateFuncs[name] = f
}

func New(dir string, assetsPrefix string) *Template {
	t := &Template{
		Template:     template.New(""),
		Dir:          dir,
		AssetsPrefix: assetsPrefix,
		Trees:        make(map[string]*parse.Tree),
	}
	// This is required so text/template calls t.init()
	// and initializes the common data structure
	t.Template.New("")
	funcs := FuncMap{
		"__topAssets":    func() []assets.Asset { return t.topAssets },
		"__bottomAssets": func() []assets.Asset { return t.bottomAssets },
		"asset":          func(arg string) string { return files.StaticFileUrl(t.AssetsPrefix, arg) },
	}
	t.Funcs(funcs)
	t.Template.Funcs(template.FuncMap(funcs)).Funcs(templateFuncs)
	return t
}

func Parse(dir string, assetsDir string, file string) (*Template, error) {
	t := New(dir, assetsDir)
	err := t.Parse(file)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func MustParse(dir string, assetsDir string, file string) *Template {
	t, err := Parse(dir, assetsDir, file)
	if err != nil {
		log.Fatalf("Error loading template %s: %s\n", file, err)
	}
	return t
}

func compileTree(name, text string) *parse.Tree {
	funcs := map[string]interface{}{
		"__topAssets":    func() {},
		"__bottomAssets": func() {},
		"asset":          func() {},
	}
	treeMap, err := parse.Parse(name, text, leftDelim, rightDelim, funcs)
	if err != nil {
		panic(err)
	}
	return treeMap[name]
}
