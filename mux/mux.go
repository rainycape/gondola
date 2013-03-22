// Package mux provides a Mux implementation which does
// regexp based URL routing and provides functions for
// managing the lifecycle of a request at different
// points.
package mux

import (
	"fmt"
	"gondola/errors"
	"gondola/files"
	"gondola/template/config"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	replaceCapturesRe    = regexp.MustCompile(`\(([^\?]|\?P).+?\)`)
	replaceNonCapturesRe = regexp.MustCompile(`\(\?(?:\w+:)?(.*?)\)`)
)

type RecoverHandler func(interface{}, *Context) interface{}

// ContextProcessor functions run before the request is matched to
// a handler and might alter the context in any way they see fit
type ContextProcessor func(*Context) bool

type Handler func(*Context)

type handlerInfo struct {
	host    string
	name    string
	re      *regexp.Regexp
	handler Handler
}

type Mux struct {
	ContextProcessors []ContextProcessor
	ContextFinalizers []ContextFinalizer
	RecoverHandlers   []RecoverHandler
	handlers          []*handlerInfo
	contextTransform  *reflect.Value
	trustXHeaders     bool
	keepRemotePort    bool
}

// HandleFunc adds an anonymous handler. Anonymous handlers can't be reversed.
func (mux *Mux) HandleFunc(pattern string, handler Handler) {
	mux.HandleHostNamedFunc(pattern, handler, "", "")
}

// HandleNamedFunc adds named handler. Named handlers can be reversed using
// Mux.Reverse() or the "reverse" function in the templates.
func (mux *Mux) HandleNamedFunc(pattern string, handler Handler, name string) {
	mux.HandleHostNamedFunc(pattern, handler, "", name)
}

// HandleHostFunc works like HandleFunc(), but restricts matches to the given host.
func (mux *Mux) HandleHostFunc(pattern string, handler Handler, host string) {
	mux.HandleHostNamedFunc(pattern, handler, host, "")
}

// HandleHostNamedFunc works like HandleNamedFunc(), but restricts matches to the given host.
func (mux *Mux) HandleHostNamedFunc(pattern string, handler Handler, host string, name string) {
	info := &handlerInfo{
		host:    host,
		name:    name,
		re:      regexp.MustCompile(pattern),
		handler: handler,
	}
	mux.handlers = append(mux.handlers, info)
}

// AddContextProcessor adds context processor to the Mux.
// Context processors run in the same order they were added
// before the mux starts matching the request to a handler and
// may alter the request in any way they see fit as well as
// writing to the context. If any of the processors returns
// true, the request is considered as served and no further
// processing to it is done.
func (mux *Mux) AddContextProcessor(cp ContextProcessor) {
	mux.ContextProcessors = append(mux.ContextProcessors, cp)
}

// AddContextFinalizer adds a context finalizer to the mux.
// Context finalizers run in the order they were added and
// after the request has been served (even if it was stopped by
// a context processor). They're intended as a way to perform
// some logging or cleanup (e.g. closing database connections
// that might have been opened during the request lifetime)
func (mux *Mux) AddContextFinalizer(cf ContextFinalizer) {
	mux.ContextFinalizers = append(mux.ContextFinalizers, cf)
}

// AddRecoverHandler adds a recover handler to the mux.
// Recover handlers are executed in the order they were added in case
// there's a panic while serving a request. The handlers may write
// to the context. If any recover handler returns nil the error is
// considered as handled and no panic is raised.
func (mux *Mux) AddRecoverHandler(rh RecoverHandler) {
	mux.RecoverHandlers = append(mux.RecoverHandlers, rh)
}

// ContextTransform sets the function which transforms a *mux.Context
// into your own context type before passing it to the template
// rendering system, so you can call
// your own custom methods from the templates. See the documentation
// on mux.Context to learn how to create your own custom context methods.
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

// TrustsXHeaders returns if the mux uses X headers
// for determining the remote IP and scheme. See SetTrustXHeaders()
// for a more detailed explanation.
func (mux *Mux) TrustsXHeaders() bool {
	return mux.trustXHeaders
}

// SetTrustXHeaders sets if the mux uses X headers like
// X-Real-IP, X-Forwarded-For, X-Scheme and X-Forwarded-Proto
// to override the remote IP and scheme. This is useful
// when running your application behind a proxy or load balancer.
// The default is disabled. Please, keep in mind that enabling
// XHeaders processing when not running behind a proxy or load
// balancer which sanitizes the input *IS A SECURITY RISK*.
func (mux *Mux) SetTrustXHeaders(t bool) {
	mux.trustXHeaders = t
}

// KeepsRemotePort returns if the mux keeps the remote port
// in http.Request.RemoteAddr field. See SetKeepRemotePort
// for a more detailed explanation.
func (mux *Mux) KeepsRemotePort() bool {
	return mux.keepRemotePort
}

// SetKeepRemovePort sets if the mux keeps the remote port
// in http.Request.RemoteAddr field. Since the remote port
// is rarely useful, this defaults to false, so RemoteAddr
// will only contain an address
func (mux *Mux) SetKeepRemotePort(k bool) {
	mux.keepRemotePort = k
}

// HandleStaticFiles adds several handlers to the mux which handle
// static files efficiently and allows the use of the "assset"
// function from the templates. prefix might be a relative
// (e.g. /static/) or absolute (e.g. http://static.example.com/) url
// while dir should be the path to the directory where the static
// assets reside. You probably want to use RelativePath() in gondola/util
// to define the directory relative to the application binary. Note
// that /favicon.ico and /robots.txt will be handled too, but they
// will must be in the directory which contains the rest of the assets.
func (mux *Mux) HandleStaticFiles(prefix string, dir string) {
	filesHandler := files.StaticFilesHandler(prefix, dir)
	handler := func(ctx *Context) {
		filesHandler(ctx, ctx.R)
	}
	mux.HandleFunc(prefix, handler)
	mux.HandleFunc("^/favicon.ico$", handler)
	mux.HandleFunc("^/robots.txt$", handler)
	config.SetStaticFilesUrl(prefix)
}

// Reverse obtains the url given a handler name and its arguments.
// The number of arguments must be equal to the
// number of captured parameters in the patttern for the handler
// e.g. given the pattern ^/article/\d+/[\w\-]+/$, you should provide
// 2 arguments and passing 42 and "the-ultimate-answer-to-life-the-universe-and-everything"
// would return "/article/42/the-ultimate-answer-to-life-the-universe-and-everything/"
// If the handler is also restricted to a given hostname, the return value
// will be a scheme relative url e.g. //www.example.com/article/...
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
			format := replaceCapturesRe.ReplaceAllString(clean, "%v")
			if len(args) != strings.Count(format, "%v") {
				return "", fmt.Errorf("Handler \"%s\" requires %d arguments, %d received instead", name,
					strings.Count(format, "%v"), len(args))
			}
			/* Replace non-capturing groups with their re */
			format = replaceNonCapturesRe.ReplaceAllString(format, "$1")
			/* eg (?flags:re) */
			reversed := fmt.Sprintf(format, args...)
			if v.host != "" {
				reversed = fmt.Sprintf("//%s%s", v.host, reversed)
			}
			return reversed, nil
		}
	}
	return "", fmt.Errorf("No handler named \"%s\"", name)
}

// ListenAndServe Starts listening on all the interfaces on the given port.
// If you need more granularity you can use http.ListenAndServe() directly
// e.g.
// http.ListenAndServe("127.0.0.1:8000", mymux)
func (mux *Mux) ListenAndServe(port int) error {
	return http.ListenAndServe(":"+strconv.Itoa(port), mux)
}

func (mux *Mux) readXHeaders(r *http.Request) {
	/* TODO: Handle scheme */
	realIp := r.Header.Get("X-Real-IP")
	if realIp != "" {
		r.RemoteAddr = realIp
		return
	}
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		r.RemoteAddr = forwardedFor
	}
}

func (mux *Mux) recover(ctx *Context) {
	if err := recover(); err != nil {
		for _, v := range mux.RecoverHandlers {
			err = v(err, ctx)
			if err == nil {
				break
			}
		}
		if err != nil {
			if gerr, ok := err.(errors.Error); ok {
				ctx.WriteHeader(gerr.StatusCode())
				ctx.Write([]byte(gerr.String()))
				err = nil
			}
		}
		if err != nil {
			panic(err)
		}
	}
}

// ServeHTTP is called from the net/http system. You shouldn't need
// to call this function
func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := &Context{ResponseWriter: w, R: r, mux: mux}
	defer mux.closeContext(ctx)
	defer mux.recover(ctx)
	if mux.trustXHeaders {
		mux.readXHeaders(r)
	}
	if !mux.keepRemotePort {
		addr := r.RemoteAddr
		if strings.Count(addr, ".") == 3 {
			/* IPv4 e.g. 127.0.0.1:8000 */
			idx := strings.Index(addr, ":")
			if idx >= 0 {
				r.RemoteAddr = addr[:idx]
			}
		} else {
			/* IPv6 e.g. [1fff:0:a88:85a3::ac1f]:8001 */
			if addr != "" && addr[0] == '[' {
				idx := strings.Index(addr, "]")
				if idx >= 0 {
					r.RemoteAddr = addr[1:idx]
				}
			}
		}
	}
	for _, v := range mux.ContextProcessors {
		if v(ctx) {
			return
		}
	}

	/* Mux handlers first */
	for _, v := range mux.handlers {
		if v.host != "" && v.host != r.Host {
			continue
		}
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
			v.handler(ctx)
			break
		}
	}
	/* Not found */
}

func (mux *Mux) closeContext(ctx *Context) {
	for _, v := range mux.ContextFinalizers {
		v(ctx)
	}
	ctx.Close()
}

// Returns a new Mux initialized with the default values
func New() *Mux {
	return &Mux{}
}
