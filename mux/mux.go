package mux

import (
	"net/http"
	"regexp"
)

type RecoverHandler func(*http.Request, http.ResponseWriter, interface{}) interface{}

type RequestProcessor func(*http.Request, http.ResponseWriter, *Context) (*http.Request, bool)

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

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			for _, v := range mux.RecoverHandlers {
				err = v(r, w, err)
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
	ctx := &Context{}
	for _, v := range mux.RequestProcessors {
		r, stop = v(r, w, ctx)
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

func New() *Mux {
	return &Mux{}
}

type Context struct {
	submatches []string
	params     map[string]string
}

func (c *Context) Count() int {
	return len(c.submatches)
}

func (c *Context) IndexValue(idx int) string {
	if idx < len(c.submatches) {
		return c.submatches[idx]
	}
	return ""
}

func (c *Context) ParamValue(name string) string {
	return c.params[name]
}
