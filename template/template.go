package template

import (
	"bytes"
	"gondola/mux"
	"gondola/template/config"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
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
	commentRe = regexp.MustCompile(`(?s:\{\{\\*(.*?)\*/\}\})`)
	keyRe     = regexp.MustCompile(`(?s:\s*([\w\-_])+:)`)
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
	return config.StaticFilesUrl()
}

// Sets the base url for static assets. Might be
// relative (e.g. /static/) or absolute (e.g. http://static.example.com/)
func SetStaticFilesUrl(url string) {
	config.SetStaticFilesUrl(url)
}

func Path() string {
	return config.Path()
}

// Sets the path for the template files. By default, it's
// initialized to the directory tmpl relative to the executable
func SetPath(p string) {
	config.SetPath(p)
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
	context *mux.Context
}

func (t *Template) parseScripts(value string, st ScriptType) {
	for _, v := range strings.Split(value, ",") {
		name := strings.TrimSpace(v)
		t.scripts = append(t.scripts, &script{name, st})
	}
}

func (t *Template) Execute(w io.Writer, data interface{}) error {
	var buf bytes.Buffer
	ctx, _ := w.(*mux.Context)
	t.mu.Lock()
	t.context = ctx
	err := t.ExecuteTemplate(&buf, t.root, data)
	t.mu.Unlock()
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
		if rw, ok := w.(http.ResponseWriter); ok {
			http.Error(rw, "Error", http.StatusInternalServerError)
		}
		log.Panicf("Error executing template: %s\n", err)
	}
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
	return path.Join(config.Path(), name)
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
		"Ctx":     makeContext(t),
		"reverse": makeReverse(t),
		"request": makeRequest(t),
	})
	tmpl, err = tmpl.Parse(s)
	if err != nil {
		return err
	}
	return nil
}

func Parse(name string) (*Template, error) {
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

func MustParse(name string) *Template {
	t, err := Parse(name)
	if err != nil {
		log.Fatalf("Error loading template %s: %s\n", name, err)
	}
	return t
}

func Execute(name string, w io.Writer, data interface{}) error {
	t, err := Parse(name)
	if err != nil {
		return err
	}
	return t.Execute(w, data)
}

func MustExecute(name string, w io.Writer, data interface{}) {
	err := Execute(name, w, data)
	if err != nil {
		log.Panic(err)
	}
}
