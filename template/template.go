package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gondola/files"
	"html/template"
	"io/ioutil"
	log "logging"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"text/template/parse"
)

type ScriptType int

const (
	_ ScriptType = iota
	ScriptTypeStandard
	ScriptTypeAsync
	ScriptTypeOnload
)

var (
	staticFilesUrl string
	templatesPath  string
	templates      = make(map[string]*Template)
	mutex          = &sync.RWMutex{}
	debug          = false
)

var assetsBoilerPlate = `
{{ define "stylesheets" }}
  {{ range .__styles }}
    <link rel="stylesheet" type="text/css" href="{{ . }}">
  {{ end }}
{{ end }}
{{ define "scripts" }}
  {{ range .__scripts }}
    {{ if .IsAsync }}
      <script type="text/javascript">
        (function() {
          var li = document.createElement('script'); li.type = 'text/javascript'; li.async = true;
          li.src = "{{ .Url }}";
          var s = document.getElementsByTagName('script')[0]; s.parentNode.insertBefore(li, s);
        })();
      </script>
    {{ else }}
      <script type="text/javascript" src="{{ .Url }}"></script>
    {{ end }}
  {{ end }}
{{ end }}
`

func StaticFilesUrl() string {
	return staticFilesUrl
}

func SetStaticFilesUrl(url string) {
	staticFilesUrl = url
}

func TemplatesPath() string {
	return templatesPath
}

func SetTemplatesPath(p string) {
	templatesPath = p
}

func SetDebug(d bool) {
	debug = d
}

type Script struct {
	Url  template.JS
	Type ScriptType
}

func (s *Script) IsAsync() bool {
	return s.Type == ScriptTypeAsync
}

type Template struct {
	*template.Template
	Filenames   []string
	Scripts     []Script
	Stylesheets []string
}

func (t *Template) Clone() (*Template, error) {
	c, err := t.Template.Clone()
	if err != nil {
		return nil, err
	}
	clon := &Template{Template: c}
	clon.Filenames = t.Filenames
	clon.Scripts = t.Scripts
	clon.Stylesheets = t.Stylesheets
	return clon, nil
}

func (t *Template) walkNodes(nodes []parse.Node, nodeType parse.NodeType, f func(node parse.Node)) {
	for _, node := range nodes {
		nt := node.Type()
		if nt == nodeType {
			f(node)
		}
		switch node.Type() {
		case parse.NodeIf:
			ifNode := node.(*parse.IfNode)
			t.walkNodes(ifNode.List.Nodes, nodeType, f)
			if ifNode.ElseList != nil {
				t.walkNodes(ifNode.ElseList.Nodes, nodeType, f)
			}
		case parse.NodeRange:
			rangeNode := node.(*parse.RangeNode)
			t.walkNodes(rangeNode.List.Nodes, nodeType, f)
			if rangeNode.ElseList != nil {
				t.walkNodes(rangeNode.ElseList.Nodes, nodeType, f)
			}
		case parse.NodeWith:
			withNode := node.(*parse.WithNode)
			t.walkNodes(withNode.List.Nodes, nodeType, f)
			if withNode.ElseList != nil {
				t.walkNodes(withNode.ElseList.Nodes, nodeType, f)
			}
		}
	}
}

func (t *Template) appendTemplateNode(node parse.Node) {
	parts := strings.Split(node.String(), "\"")
	name := parts[1]
	if strings.HasSuffix(name, ".html") {
		filename := GetTemplatePath(name)
		_, err := TemplateFromFile(t.Template, name, filename)
		if err != nil {
			panic(err)
		}
		t.Filenames = append(t.Filenames, filename)
	}
}

func (t *Template) ParseRemainingTemplates() {
	for _, v := range t.Filenames {
		b, _ := ioutil.ReadFile(v)
		tree, err := parse.Parse(v, string(b), "", "", templateFuncs)
		if err != nil {
			panic(err)
		}
		t.walkNodes(tree[v].Root.Nodes, parse.NodeTemplate, func(node parse.Node) {
			t.appendTemplateNode(node)
		})
	}
}

func (t *Template) appendStaticAsset(node parse.Node) {
	str := node.String()
	if strings.HasPrefix(str, "{{stylesheet") {
		t.AddStylesheet(node)
	} else if strings.HasPrefix(str, "{{script") {
		t.AddScript(node, ScriptTypeStandard)
	} else if strings.HasPrefix(str, "{{ascript") {
		t.AddScript(node, ScriptTypeAsync)
	}
}

func (t *Template) ParseStaticAssets() {
	for _, v := range t.Filenames {
		b, _ := ioutil.ReadFile(v)
		tree, err := parse.Parse(v, string(b), "", "", templateFuncs)
		if err != nil {
			panic(err)
		}
		t.walkNodes(tree[v].Root.Nodes, parse.NodeAction, func(node parse.Node) {
			t.appendStaticAsset(node)
		})
	}
}

func (t *Template) getAssetUrl(node parse.Node) string {
	str := node.String()
	parts := strings.Split(str, "\"")
	name := parts[1]
	var url string
	if strings.HasPrefix(name, "//") || strings.Contains(name, "://") {
		url = name
	} else {
		url = GetAssetUrl(name)
	}
	return url
}

func (t *Template) addAssetNode(node parse.Node, nodes *[]string) {
	str := node.String()
	parts := strings.Split(str, "\"")
	name := parts[1]
	var url string
	if strings.HasPrefix(name, "//") {
		url = name
	} else {
		url = GetAssetUrl(name)
	}
	*nodes = append(*nodes, url)
}

func (t *Template) AddScript(node parse.Node, scriptType ScriptType) {
	url := t.getAssetUrl(node)
	t.Scripts = append(t.Scripts, Script{template.JS(url), scriptType})
}

func (t *Template) AddStylesheet(node parse.Node) {
	url := t.getAssetUrl(node)
	t.Stylesheets = append(t.Stylesheets, url)
}

func GetAssetUrl(name ...string) string {
	n := strings.Join(name, "")
	return files.StaticFileUrl(staticFilesUrl, n)
}

func eq(args ...interface{}) bool {
	if len(args) == 0 {
		return false
	}
	x := args[0]
	switch x := x.(type) {
	case string, int, int64, byte, float32, float64:
		for _, y := range args[1:] {
			if x == y {
				return true
			}
		}
		return false
	}

	for _, y := range args[1:] {
		if reflect.DeepEqual(x, y) {
			return true
		}
	}
	return false
}

func neq(args ...interface{}) bool {
	return !eq(args...)
}

func _json(arg interface{}) string {
	if arg == nil {
		return ""
	}
	b, err := json.Marshal(arg)
	if err == nil {
		return string(b)
	}
	return ""
}

func nz(x interface{}) bool {
	switch x := x.(type) {
	case int, uint, int64, uint64, byte, float32, float64:
		if x != 0 {
			return true
		}
	}
	return false
}

func lower(x string) string {
	return strings.ToLower(x)
}

func join(x []string, sep string) string {
	s := ""
	for _, v := range x {
		s += fmt.Sprintf("%v%s", v, sep)
	}
	if len(s) > 0 {
		return s[:len(s)-len(sep)]
	}
	return ""
}

func chunks(items interface{}, length int) [][]interface{} {
	var chunks [][]interface{}
	var chunk []interface{}
	for _, v := range items.([]interface{}) {
		chunk = append(chunk, v)
		if len(chunk) == length {
			chunks = append(chunks, chunk)
			chunk = nil
		}
	}
	if len(chunk) > 0 {
		chunks = append(chunks, chunk)
	}
	return chunks
}

func makemap(args ...interface{}) map[string]interface{} {
	items := make(map[string]interface{})
	var key = ""
	for _, v := range args {
		if key != "" {
			items[key] = v
			key = ""
		} else {
			key = v.(string)
		}
	}
	return items
}

func dummy(arg string) string {
	return ""
}

func i18n(arg string) string {
	return arg
}

var templateFuncs template.FuncMap = template.FuncMap{
	"asset":      GetAssetUrl,
	"eq":         eq,
	"neq":        neq,
	"json":       _json,
	"nz":         nz,
	"lower":      lower,
	"join":       join,
	"chunks":     chunks,
	"makemap":    makemap,
	"stylesheet": dummy,
	"script":     dummy,
	"ascript":    dummy,
	"_":          i18n,
}

func AddFunc(name string, f interface{}) {
	templateFuncs[name] = f
}

func GetTemplatePath(basename string) string {
	return path.Join(templatesPath, basename)
}

func TemplateFromFile(t *template.Template, name string, filename string) (*template.Template, error) {
	var tmpl *template.Template
	if t != nil {
		tmpl = t.New(name)
	} else {
		tmpl = template.New(name)
	}
	tmpl = tmpl.Funcs(templateFuncs)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	s := string(b)
	if idx := strings.Index(s, "</head>"); idx >= 0 {
		s = s[:idx] + "{{ template \"stylesheets\" . }}" + s[idx:]
	}
	if idx := strings.Index(s, "</body>"); idx >= 0 {
		s = s[:idx] + "{{ template \"scripts\" . }}" + s[idx:]
	}
	if t == nil {
		s += assetsBoilerPlate
	}
	tmpl, err = tmpl.Parse(s)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

func GetTemplateChain(name ...string) *Template {
	var t *template.Template = nil
	var root *Template = nil
	for _, v := range name {
		basename := fmt.Sprintf("%s.html", v)
		templatePath := GetTemplatePath(basename)
		var err error
		if t == nil {
			t, err = TemplateFromFile(nil, "base", templatePath)
			if err != nil {
				panic(err)
			}
			root = &Template{Template: t}
		} else {
			contentsName := fmt.Sprintf("%s_contents", t.Name())
			t, err = TemplateFromFile(t, contentsName, templatePath)
			if err != nil {
				panic(err)
			}
		}
		root.Filenames = append(root.Filenames, templatePath)
	}
	return root
}

func Render(w http.ResponseWriter, r *http.Request, templateNames []string, data map[string]interface{}) {
	key := strings.Join(templateNames, "/")
	mutex.RLock()
	t := templates[key]
	mutex.RUnlock()
	if t == nil {
		t = GetTemplateChain(templateNames...)
		t.ParseRemainingTemplates()
		t.ParseStaticAssets()
		if !debug {
			mutex.Lock()
			templates[key] = t
			mutex.Unlock()
		}
	}
	data["__styles"] = &t.Stylesheets
	data["__scripts"] = &t.Scripts
	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		log.Errorf("Error executing template: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	header := w.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Write(buf.Bytes())
}
