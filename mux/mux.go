package mux

import (
	"gondola/cache"
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
	ctx := &Context{}
	defer ctx.Close()
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

func New() *Mux {
	return &Mux{}
}

type Context struct {
	submatches []string
	params     map[string]string
	c          *cache.Cache
	Data       interface{} /* Left to the user */
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

func (c *Context) Cache() *cache.Cache {
	if c.c == nil {
		c.c = cache.NewDefault()
	}
	return c.c
}

func (c *Context) Close() {
	if c.c != nil {
		c.c.Close()
		c.c = nil
	}
}
