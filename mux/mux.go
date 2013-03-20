package mux

import (
	"fmt"
	"gondola/files"
	"gondola/template/config"
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

type RecoverHandler func(http.ResponseWriter, *http.Request, interface{}) interface{}

type RequestProcessor func(http.ResponseWriter, *http.Request, *Context) (*http.Request, bool)

type Handler func(http.ResponseWriter, *http.Request, *Context)

type handlerInfo struct {
	/* TODO: Add support for patterns specifing a host */
	name    string
	re      *regexp.Regexp
	handler Handler
}

type Mux struct {
	handlers          []*handlerInfo
	RequestProcessors []RequestProcessor
	RecoverHandlers   []RecoverHandler
	ContextFinalizers []ContextFinalizer
	contextTransform  *reflect.Value
}

func (mux *Mux) HandleFunc(pattern string, handler Handler) {
	mux.HandleNamedFunc(pattern, handler, "")
}

func (mux *Mux) HandleNamedFunc(pattern string, handler Handler, name string) {
	info := &handlerInfo{
		name:    name,
		re:      regexp.MustCompile(pattern),
		handler: handler,
	}
	mux.handlers = append(mux.handlers, info)
}

func (mux *Mux) AddRequestProcessor(rp RequestProcessor) {
	mux.RequestProcessors = append(mux.RequestProcessors, rp)
}

func (mux *Mux) AddRecoverHandler(rh RecoverHandler) {
	mux.RecoverHandlers = append(mux.RecoverHandlers, rh)
}

func (mux *Mux) AddContextFinalizer(cf ContextFinalizer) {
	mux.ContextFinalizers = append(mux.ContextFinalizers, cf)
}

func (mux *Mux) SetContextTransform(f interface{}) {
	t := reflect.TypeOf(f)
	if t.Kind() != reflect.Func {
		panic(fmt.Errorf("Context transform must be a function, instead it's %t", f))
	}
	if t.IsVariadic() {
		panic(fmt.Errorf("Context transform can't be a variadic function"))
	}
	contextType := reflect.TypeOf(&Context{})
	if t.NumIn() != 1 || t.In(0) != contextType {
		panic(fmt.Errorf("Context transform must receive only 1 %s argument", contextType))
	}
	if t.NumOut() != 1 || t.Out(0).Kind() != reflect.Ptr || t.Out(0).Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("Context transform must return just 1 argument which must be a pointer to a struct"))
	}
	/* All checks passed */
	val := reflect.ValueOf(f)
	mux.contextTransform = &val
}

func (mux *Mux) HandleStaticFiles(prefix string, dir string) {
	filesHandler := files.StaticFilesHandler(prefix, dir)
	handler := func(w http.ResponseWriter, r *http.Request, ctx *Context) {
		filesHandler(w, r)
	}
	mux.HandleFunc(prefix, handler)
	mux.HandleFunc("^/favicon.ico$", handler)
	mux.HandleFunc("^/robots.txt$", handler)
	config.SetStaticFilesUrl(prefix)
}

func (mux *Mux) Reverse(name string, args ...interface{}) (string, error) {
	if name == "" {
		return "", fmt.Errorf("No handler name specified")
	}
	for _, v := range mux.handlers {
		if v.name == name {
			pattern := v.re.String()
			clean := strings.Trim(pattern, "^$")
			/* Replace capturing groups with a format specifier */
			/* e.g. (re) and (?P<name>re) */
			format := regexp.MustCompile(`\(([^\?]|\?P).+?\)`).ReplaceAllString(clean, "%v")
			if len(args) != strings.Count(format, "%v") {
				return "", fmt.Errorf("Handler \"%s\" requires %d arguments, %d received instead", name,
					strings.Count(format, "%v"), len(args))
			}
			/* Replace non-capturing groups with their re */
			format = regexp.MustCompile(`\(\?(?:\w+:)?(.*?)\)`).ReplaceAllString(format, "$1")
			/* eg (?flags:re) */
			return fmt.Sprintf(format, args...), nil
		}
	}
	return "", fmt.Errorf("No handler named \"%s\"", name)
}

func (mux *Mux) recover(w http.ResponseWriter, r *http.Request) {
	if err := recover(); err != nil {
		for _, v := range mux.RecoverHandlers {
			err = v(w, r, err)
			if err == nil {
				break
			}
		}
		if err != nil {
			panic(err)
		}
	}
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer mux.recover(w, r)
	stop := false
	ctx := &Context{W: w, R: r, mux: mux}
	defer mux.closeContext(ctx)
	for _, v := range mux.RequestProcessors {
		r, stop = v(w, r, ctx)
		if stop {
			break
		}
	}
	if !stop {
		/* Try mux handlers first */
		for _, v := range mux.handlers {
			if submatches := v.re.FindStringSubmatch(r.URL.Path); submatches != nil {
				params := map[string]string{}
				for ii, n := range v.re.SubexpNames() {
					if n != "" {
						params[n] = submatches[ii]
					}
				}
				ctx.submatches = submatches
				ctx.params = params
				ctx.handlerName = v.name
				v.handler(w, r, ctx)
				stop = true
				break
			}
		}
		/* Not found */
	}
}

func (mux *Mux) closeContext(ctx *Context) {
	for _, v := range mux.ContextFinalizers {
		v(ctx)
	}
	ctx.Close()
}

func New() *Mux {
	return &Mux{}
}
