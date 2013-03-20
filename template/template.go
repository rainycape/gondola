package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gondola/files"
	"gondola/util"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
	templatesPath  = util.RelativePath("tmpl")
	commentRe      = regexp.MustCompile(`(?s:\{\{\\*(.*?)\*/\}\})`)
	keyRe          = regexp.MustCompile(`(?s:\s*([\w\-_])+:)`)
)

var stylesBoilerplate = `
  {{ range _getstyles }}
    <link rel="stylesheet" type="text/css" href="{{ asset . }}">
  {{ end }}
`

var scriptsBoilerplate = `
  {{ range _getscripts }}
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

func StaticFilesUrl() string {
	return staticFilesUrl
}

func SetStaticFilesUrl(url string) {
	staticFilesUrl = url
}

func Path() string {
	return templatesPath
}

func SetPath(p string) {
	templatesPath = p
}

type script struct {
	Name string
	Type ScriptType
}

func (s *script) IsAsync() bool {
	return s.Type == ScriptTypeAsync
}

type Template struct {
	*template.Template
	root    string
	scripts []*script
	styles  []string
	mu      *sync.Mutex
	context interface{}
}

func (t *Template) parseScripts(value string, st ScriptType) {
	for _, v := range strings.Split(value, ",") {
		name := strings.TrimSpace(v)
		t.scripts = append(t.scripts, &script{name, st})
	}
}

func (t *Template) Render(w http.ResponseWriter, ctx interface{}, data interface{}) error {
	var buf bytes.Buffer
	t.mu.Lock()
	t.context = ctx
	err := t.ExecuteTemplate(&buf, t.root, data)
	t.mu.Unlock()
	if err != nil {
		return err
	}
	header := w.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Write(buf.Bytes())
	return nil
}

func (t *Template) MustRender(w http.ResponseWriter, ctx interface{}, data interface{}) {
	err := t.Render(w, ctx, data)
	if err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		log.Panicf("Error executing template: %s\n", err)
	}
}

func AssetUrl(name ...string) string {
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

var templateFuncs template.FuncMap = template.FuncMap{
	"asset": AssetUrl,
	"eq":    eq,
	"neq":   neq,
	"json":  _json,
	"nz":    nz,
	"lower": lower,
	"join":  join,
}

func AddFunc(name string, f interface{}) {
	templateFuncs[name] = f
}

func parseComment(value string, t *Template, name string) {
	lines := strings.Split(value, "\n")
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
				case "script":
					fallthrough
				case "scripts":
					t.parseScripts(value, ScriptTypeStandard)
				case "ascript":
					fallthrough
				case "ascripts":
					t.parseScripts(value, ScriptTypeAsync)
				case "css":
					fallthrough
				case "styles":
					for _, v := range strings.Split(value, ",") {
						style := strings.TrimSpace(v)
						t.styles = append(t.styles, style)
					}
				case "extend":
					fallthrough
				case "extends":
					load(value, t)
					extended = true
				}
			}
		}
	}
	if !extended {
		t.root = name
	}
}

func getTemplatePath(name string) string {
	return path.Join(templatesPath, name)
}

func load(name string, t *Template) error {
	f := getTemplatePath(name)
	b, err := ioutil.ReadFile(f)
	if err != nil {
		return err
	}
	s := string(b)
	matches := commentRe.FindStringSubmatch(s)
	comment := ""
	if matches != nil && len(matches) > 0 {
		comment = matches[1]
	}
	parseComment(comment, t, name)
	if idx := strings.Index(s, "</head>"); idx >= 0 {
		s = s[:idx] + "{{ template \"__styles\" }}" + s[idx:]
	}
	if idx := strings.Index(s, "</body>"); idx >= 0 {
		s = s[:idx] + "{{ template \"__scripts\" }}" + s[idx:]
	}
	var tmpl *template.Template
	if t.Template == nil {
		t.Template = template.New(name)
		tmpl = t.Template
	} else {
		tmpl = t.Template.New(name)
	}
	tmpl = tmpl.Funcs(templateFuncs)
	tmpl = tmpl.Funcs(template.FuncMap{
		"Context": func() interface{} {
			return t.context
		},
	})
	tmpl, err = tmpl.Parse(s)
	if err != nil {
		return err
	}
	return nil
}

func Load(name string) (*Template, error) {
	t := &Template{}
	t.mu = &sync.Mutex{}
	err := load(name, t)
	if err != nil {
		return nil, err
	}
	/* Add styles and scripts */
	styles := t.Template.New("__styles")
	styles.Funcs(template.FuncMap{
		"_getstyles": func() []string { return t.styles },
	})
	styles.Parse(stylesBoilerplate)
	scripts := t.Template.New("__scripts")
	scripts.Funcs(template.FuncMap{
		"_getscripts": func() []*script { return t.scripts },
	})
	scripts.Parse(scriptsBoilerplate)
	return t, nil
}

func MustLoad(name string) *Template {
	t, err := Load(name)
	if err != nil {
		log.Fatalf("Error loading template %s: %s\n", name, err)
	}
	return t
}

func Render(name string, w http.ResponseWriter, ctx interface{}, data interface{}) error {
	t, err := Load(name)
	if err != nil {
		return err
	}
	return t.Render(w, ctx, data)
}

func MustRender(name string, w http.ResponseWriter, ctx interface{}, data interface{}) {
	err := Render(name, w, ctx, data)
	if err != nil {
		log.Panic(err)
	}
}
