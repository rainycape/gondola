package template

import (
	"bytes"
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

type ScriptType int

const (
	_ ScriptType = iota
	ScriptTypeStandard
	ScriptTypeAsync
	ScriptTypeOnload
)

const (
	leftDelim       = "{{"
	rightDelim      = "}}"
	stylesTmplName  = "__styles"
	scriptsTmplName = "__scripts"
)

var stylesBoilerplate = `
  {{ range __getstyles }}
    <link rel="stylesheet" type="text/css" href="{{ asset . }}">
  {{ end }}
`

var scriptsBoilerplate = `
  {{ range __getscripts }}
    {{ if .IsAsync }}
      <script type="text/javascript">
        (function() {
          var li = document.createElement('script'); li.type = 'text/javascript'; li.async = true;
          li.src = "{{ asset .Name }}";
          var s = document.getElementsByTagName('script')[0]; s.parentNode.insertBefore(li, s);
        })();
      </script>
    {{ else }}
      <script type="text/javascript" src="{{ asset .Name }}"></script>
    {{ end }}
  {{ end }}
`

var (
	commentRe   = regexp.MustCompile(`(?s:\{\{\\*(.*?)\*/\}\})`)
	keyRe       = regexp.MustCompile(`(?s:\s*([\w\-_])+:)`)
	stylesTree  = compileTree(stylesTmplName, stylesBoilerplate)
	scriptsTree = compileTree(scriptsTmplName, scriptsBoilerplate)
)

type script struct {
	Name string
	Type ScriptType
}

func (s *script) IsAsync() bool {
	return s.Type == ScriptTypeAsync
}

type Template struct {
	*template.Template
	funcMap FuncMap
	root    string
	scripts []*script
	styles  []string
}

func (t *Template) parseScripts(value string, st ScriptType) {
	for _, v := range strings.Split(value, ",") {
		name := strings.TrimSpace(v)
		t.scripts = append(t.scripts, &script{name, st})
	}
}

func (t *Template) parseComment(comment string, file string) error {
	lines := strings.Split(comment, "\n")
	extended := false
	for _, v := range lines {
		m := keyRe.FindStringSubmatchIndex(v)
		if m != nil && m[0] == 0 && len(m) == 4 {
			start := m[1] - m[3]
			end := start + m[2]
			key := strings.TrimSpace(v[start:end])
			value := strings.TrimSpace(v[m[1]:])
			if value != "" {
				switch strings.ToLower(key) {
				case "script", "scripts":
					t.parseScripts(value, ScriptTypeStandard)
				case "ascript", "ascripts":
					t.parseScripts(value, ScriptTypeAsync)
				case "css", "style", "styles":
					for _, v := range strings.Split(value, ",") {
						style := strings.TrimSpace(v)
						t.styles = append(t.styles, style)
					}
				case "extend", "extends":
					extendedFile := path.Join(path.Dir(file), value)
					err := t.load(extendedFile)
					if err != nil {
						return err
					}
					extended = true
				}
			}
		}
	}
	if !extended {
		t.root = file
	}
	return nil
}

func (t *Template) load(file string) error {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	s := string(b)
	matches := commentRe.FindStringSubmatch(s)
	comment := ""
	if matches != nil && len(matches) > 0 {
		comment = matches[1]
	}
	err = t.parseComment(comment, file)
	if err != nil {
		return err
	}
	if idx := strings.Index(s, "</head>"); idx >= 0 {
		s = s[:idx] + "{{ template \"__styles\" }}" + s[idx:]
	}
	if idx := strings.Index(s, "</body>"); idx >= 0 {
		s = s[:idx] + "{{ template \"__scripts\" }}" + s[idx:]
	}
	var tmpl *template.Template
	if t.Template == nil {
		t.Template = template.New(file)
		tmpl = t.Template
	} else {
		tmpl = t.Template.New(file)
	}
	tmpl.Funcs(templateFuncs).Funcs(template.FuncMap(t.funcMap))
	tmpl, err = tmpl.Parse(s)
	if err != nil {
		return err
	}
	return nil
}

func (t *Template) Funcs(funcs FuncMap) {
	if t.funcMap == nil {
		t.funcMap = make(FuncMap)
	}
	for k, v := range funcs {
		t.funcMap[k] = v
	}
}

func (t *Template) Parse(file string) error {
	err := t.load(file)
	if err != nil {
		return err
	}
	/* Add styles and scripts */
	_, err = t.AddParseTree(stylesTmplName, stylesTree)
	if err != nil {
		return err
	}
	_, err = t.AddParseTree(scriptsTmplName, scriptsTree)
	if err != nil {
		return err
	}
	return nil
}

func (t *Template) Execute(w io.Writer, data interface{}) error {
	var buf bytes.Buffer
	err := t.ExecuteTemplate(&buf, t.root, data)
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

func New() *Template {
	t := &Template{}
	t.Funcs(FuncMap{
		"__getstyles":  func() []string { return t.styles },
		"__getscripts": func() []*script { return t.scripts },
	})
	return t
}

func Parse(file string) (*Template, error) {
	t := New()
	err := t.Parse(file)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func MustParse(file string) *Template {
	t, err := Parse(file)
	if err != nil {
		log.Fatalf("Error loading template %s: %s\n", file, err)
	}
	return t
}

func compileTree(name, text string) *parse.Tree {
	funcs := map[string]interface{}{
		"__getstyles":  func() {},
		"__getscripts": func() {},
		"asset":        func() {},
	}
	treeMap, err := parse.Parse(name, text, leftDelim, rightDelim, funcs)
	if err != nil {
		panic(err)
	}
	return treeMap[name]
}
