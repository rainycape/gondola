package mux

import (
	"gondola/files"
	"gondola/template"
	"net/http"
	"regexp"
)

type RecoverHandler func(http.ResponseWriter, *http.Request, interface{}) interface{}

type RequestProcessor func(http.ResponseWriter, *http.Request, *Context) (*http.Request, bool)

type Handler func(http.ResponseWriter, *http.Request, *Context)

type handlerInfo struct {
	/* TODO: Add support for patterns specifing a host */
	re      *regexp.Regexp
	handler Handler
}

type Mux struct {
	handlers          []*handlerInfo
	RequestProcessors []RequestProcessor
	RecoverHandlers   []RecoverHandler
	ContextFinalizers []ContextFinalizer
}

func (mux *Mux) HandleMuxFunc(pattern string, handler Handler) {
	info := &handlerInfo{
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

func (mux *Mux) HandleStaticFiles(prefix string, dir string) {
	filesHandler := files.StaticFilesHandler(prefix, dir)
	handler := func(w http.ResponseWriter, r *http.Request, ctx *Context) {
		filesHandler(w, r)
	}
	mux.HandleMuxFunc(prefix, handler)
	mux.HandleMuxFunc("/favicon.ico$", handler)
	mux.HandleMuxFunc("/robots.txt$", handler)
	template.SetStaticFilesUrl(prefix)
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
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
	}()
	stop := false
	ctx := &Context{W: w, R: r}
	defer mux.CloseContext(ctx)
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
				v.handler(w, r, ctx)
				stop = true
				break
			}
		}
		if !stop {
			http.DefaultServeMux.ServeHTTP(w, r)
		}
	}
}

func (mux *Mux) CloseContext(ctx *Context) {
	for _, v := range mux.ContextFinalizers {
		v(ctx)
	}
	ctx.Close()
}

func New() *Mux {
	return &Mux{}
}
