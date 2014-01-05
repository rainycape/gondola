package app

import (
	"fmt"
	"gnd.la/app/cookies"
	"gnd.la/app/serialize"
	"gnd.la/blobstore"
	"gnd.la/form/input"
	"gnd.la/i18n/table"
	"gnd.la/users"
	"gnd.la/util"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type ContextFinalizer func(*Context)

type Context struct {
	http.ResponseWriter
	R               *http.Request
	provider        ContextProvider
	reProvider      *regexpProvider
	cached          bool
	fromCache       bool
	handlerName     string
	app             *App
	statusCode      int
	started         time.Time
	cookies         *cookies.Cookies
	user            users.User
	translations    *table.Table
	hasTranslations bool
	background      bool
	// Used to store data for debugging purposes. Only
	// used when app.debug is true.
	debugStorage map[string]interface{}
	// Left to users, to store data in their custom
	// context types.
	Data interface{}
}

func (c *Context) reset() {
	c.ResponseWriter = nil
	c.R = nil
	c.cached = false
	c.fromCache = false
	c.statusCode = 0
	c.started = time.Now()
	c.cookies = nil
	c.user = nil
	c.translations = nil
	c.hasTranslations = false
	c.debugStorage = nil
	c.Data = nil
}

// Count returns the number of elements captured
// by the pattern which matched the handler.
func (c *Context) Count() int {
	return c.provider.Count()
}

// IndexValue returns the captured parameter
// at the given index or an empty string if no
// such parameter exists. Pass -1 to obtain
// the whole match.
func (c *Context) IndexValue(idx int) string {
	return c.provider.Arg(idx)
}

// RequireIndexValue works like IndexValue, but raises
// a MissingParameter error if the value is not present
// or empty.
func (c *Context) RequireIndexValue(idx int) string {
	val := c.IndexValue(idx)
	if val == "" {
		MissingParameter(fmt.Sprintf("at index %d", idx))
	}
	return val
}

// ParseIndexValue uses the captured parameter
// at the given index and tries to parse it
// into the given argument. See ParseFormValue
// for examples as well as the supported types.
func (c *Context) ParseIndexValue(idx int, arg interface{}) bool {
	val := c.IndexValue(idx)
	return c.parseTypedValue(val, arg)
}

// MustParseIndexValue works like ParseIndexValue but raises a
// MissingParameterError if the parameter is missing or an
// InvalidParameterTypeError if the parameter does not have the
// required type
func (c *Context) MustParseIndexValue(idx int, arg interface{}) {
	val := c.RequireIndexValue(idx)
	c.mustParseValue("", idx, val, arg)
}

// ParamValue returns the named captured parameter
// with the given name or an empty string if it
// does not exist.
func (c *Context) ParamValue(name string) string {
	return c.provider.Param(name)
}

// ParseParamValue uses the named captured parameter
// with the given name and tries to parse it into
// the given argument. See ParseFormValue
// for examples as well as the supported types.
func (c *Context) ParseParamValue(name string, arg interface{}) bool {
	val := c.ParamValue(name)
	return c.parseTypedValue(val, arg)
}

// FormValue returns the result of performing
// FormValue on the incoming request and trims
// any whitespaces on both sides. See the
// documentation for net/http for more details.
func (c *Context) FormValue(name string) string {
	if c.R != nil {
		return strings.TrimSpace(c.R.FormValue(name))
	}
	return ""
}

// RequireFormValue works like FormValue, but raises
// a MissingParameter error if the value is not present
// or empty.
func (c *Context) RequireFormValue(name string) string {
	val := c.FormValue(name)
	if val == "" {
		MissingParameter(name)
	}
	return val
}

// ParseFormValue tries to parse the named form value into the given
// arg e.g.
// var f float32
// ctx.ParseFormValue("quality", &f)
// var width uint
// ctx.ParseFormValue("width", &width)
// Supported types are: bool, u?int(8|16|32|64)? and float(32|64)
// Internally, ParseFormValue uses gnd.la/form/input to parse
// its arguments.
func (c *Context) ParseFormValue(name string, arg interface{}) bool {
	val := c.FormValue(name)
	return c.parseTypedValue(val, arg)
}

// MustParseFormValue works like ParseFormValue but raises a
// MissingParameterError if the parameter is missing or an
// InvalidParameterTypeError if the parameter does not have the
// required type
func (c *Context) MustParseFormValue(name string, arg interface{}) {
	val := c.RequireFormValue(name)
	c.mustParseValue(name, -1, val, arg)
}

func (c *Context) mustParseValue(name string, idx int, val string, arg interface{}) {
	if !c.parseTypedValue(val, arg) {
		t := reflect.TypeOf(arg)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if name == "" {
			name = fmt.Sprintf("at index %d", idx)
		}
		InvalidParameterType(name, t)
	}
}

// StatusCode returns the response status code. If the headers
// haven't been written yet, it returns 0
func (c *Context) StatusCode() int {
	return c.statusCode
}

func (c *Context) parseTypedValue(val string, arg interface{}) bool {
	return input.Parse(val, arg) == nil
}

// Redirect sends an HTTP redirect to the client,
// using the provided redirect, which may be either
// absolute or relative. The permanent argument
// indicates if the redirect should be sent as a
// permanent or a temporary one.
func (c *Context) Redirect(redir string, permanent bool) {
	code := http.StatusFound
	if permanent {
		code = http.StatusMovedPermanently
	}
	http.Redirect(c, c.R, redir, code)
}

// Error replies to the request with the specified
// message and HTTP code. If an error handler
// has been defined for the App, it will be
// given the opportunity to intercept the
// error and provide its own response.
func (c *Context) Error(error string, code int) {
	c.statusCode = -code
	c.app.handleHTTPError(c, error, code)
}

// NotFound is equivalent to calling Error()
// with http.StatusNotFound.
func (c *Context) NotFound(error string) {
	c.Error(error, http.StatusNotFound)
}

// Forbidden is equivalent to calling Error()
// with http.StatusForbidden.
func (c *Context) Forbidden(error string) {
	c.Error(error, http.StatusForbidden)
}

// BadRequest is equivalent to calling Error()
// with http.StatusBadRequest.
func (c *Context) BadRequest(error string) {
	c.Error(error, http.StatusBadRequest)
}

// SetCached is used internaly by cache layers.
// Don't call this method
func (c *Context) SetCached(b bool) {
	c.cached = b
}

// SetServedFromCache is used internally by cache layers.
// Don't call this method
func (c *Context) SetServedFromCache(b bool) {
	c.fromCache = b
}

// Cached() returns true if the request was
// cached by a cache layer
// (see gnd.la/cache/layer)
func (c *Context) Cached() bool {
	return c.cached
}

// ServedFromCache returns true if the request
// was served by a cache layer
// (see gnd.la/cache/layer)
func (c *Context) ServedFromCache() bool {
	return c.fromCache
}

// HandlerName returns the name of the handler which
// handled this context
func (c *Context) HandlerName() string {
	return c.handlerName
}

// App returns the App this Context originated from.
func (c *Context) App() *App {
	return c.app
}

// MustReverse calls MustReverse on the App this context originated
// from. See App.Reverse for details.
func (c *Context) MustReverse(name string, args ...interface{}) string {
	return c.app.MustReverse(name, args...)
}

// Reverse calls Reverse on the App this context originated
// from. See the App.Reverse for details.
func (c *Context) Reverse(name string, args ...interface{}) (string, error) {
	return c.app.Reverse(name, args...)
}

// RedirectReverse calls Reverse to find the URL and then sends
// the redirect to the client. See the documentation on App.Reverse
// for further details.
func (c *Context) RedirectReverse(permanent bool, name string, args ...interface{}) error {
	rev, err := c.Reverse(name, args...)
	if err != nil {
		return err
	}
	c.Redirect(rev, permanent)
	return nil
}

// MustRedirectReverse works like RedirectReverse, but panics if
// there's an error.
func (c *Context) MustRedirectReverse(permanent bool, name string, args ...interface{}) {
	err := c.RedirectReverse(permanent, name, args...)
	if err != nil {
		panic(err)
	}
}

// RedirectBack redirects the user to the previous page using
// a temporary redirect. The previous page is determined by first
// looking at the "from" GET or POST parameter (like in the sign in form)
// and then looking at the "Referer" header. If there's no previous page
// or the previous page was from another host, a redirect to / is issued.
func (c *Context) RedirectBack() {
	if c.R != nil {
		us := c.URL().String()
		redir := "/"
		// from parameter is used when redirecting to sign in page
		from := c.FormValue("from")
		if from != "" && util.EqualHosts(from, us) {
			redir = from
		} else if ref := c.R.Referer(); ref != "" && util.EqualHosts(ref, us) {
			redir = ref
		}
		// us can be protocol relative
		if us == redir || strings.HasPrefix(us, "//") && c.R.Host == urlHost(redir) {
			redir = "/"
		}
		c.Redirect(redir, false)
	}
}

// URL return the absolute URL for the current request.
func (c *Context) URL() *url.URL {
	if c.R != nil {
		u := *c.R.URL
		u.Host = c.R.Host
		if u.Scheme == "" {
			if c.R.TLS != nil {
				u.Scheme = "https"
			} else {
				u.Scheme = "http"
			}
		}
		return &u
	}
	return nil
}

// Cookies returns a coookies.Cookies object which
// can be used to set and delete cookies. See the documentation
// on gnd.la/cookies for more information.
func (c *Context) Cookies() *cookies.Cookies {
	if c.cookies == nil {
		c.cookies = cookies.New(c.R, c, c.app.Secret(),
			c.app.EncryptionKey(), c.app.DefaultCookieOptions())
	}
	return c.cookies
}

// Cache is a shorthand for ctx.App().Cache(), but panics in case
// of error, instead of returning it.
func (c *Context) Cache() *Cache {
	if c.app.c == nil {
		if c.app.debug {
			// Check if we already have a Cache
			if cache, ok := c.getDebug(debugCacheKey).(*Cache); ok {
				return cache
			}
		}
		cache, err := c.app.Cache()
		if err != nil {
			panic(err)
		}
		if c.app.debug {
			c.storeDebug(debugCacheKey, cache)
			return cache
		}
	}
	return c.app.c
}

// Blobstore is a shorthand for ctx.App().Blobstore(), but panics in
// case of error instead of returning it.
func (c *Context) Blobstore() *blobstore.Store {
	if c.app.store == nil {
		_, err := c.app.Blobstore()
		if err != nil {
			panic(err)
		}
	}
	return c.app.store
}

// Orm is a shorthand for ctx.App().Orm(), but panics in case
// of error, rather than returning it.
func (c *Context) Orm() *Orm {
	if c.app.o == nil {
		if c.app.debug {
			// Check if we already have an ORM
			if o, ok := c.getDebug(debugOrmKey).(*Orm); ok {
				return o
			}
		}
		o, err := c.app.Orm()
		if err != nil {
			panic(err)
		}
		if c.app.debug {
			c.storeDebug(debugOrmKey, o)
			return o
		}
	}
	return c.app.o
}

// Execute loads the template with the given name using the
// App template loader and executes it with the data argument.
func (c *Context) Execute(name string, data interface{}) error {
	tmpl, err := c.app.LoadTemplate(name)
	if err != nil {
		return err
	}
	return tmpl.Execute(c, data)
}

// MustExecute works like Execute, but panics if there's an error
func (c *Context) MustExecute(name string, data interface{}) {
	err := c.Execute(name, data)
	if err != nil {
		panic(err)
	}
}

// WriteJSON is equivalent to serialize.WriteJSON(ctx, data)
func (c *Context) WriteJSON(data interface{}) (int, error) {
	return serialize.WriteJSON(c, data)
}

// WriteXML is equivalent to serialize.WriteXML(ctx, data)
func (c *Context) WriteXML(data interface{}) (int, error) {
	return serialize.WriteXML(c, data)
}

// Elapsed returns the duration since this context started
// processing the request.
func (c *Context) Elapsed() time.Duration {
	return time.Since(c.started)
}

// RemoteAddress returns the remote IP without the port number.
func (c *Context) RemoteAddress() string {
	if c.R != nil {
		addr := c.R.RemoteAddr
		if addr != "" && (addr[0] == '[' || strings.IndexByte(addr, ':') > 0) {
			addr, _, _ = net.SplitHostPort(addr)
		}
		return addr
	}
	return ""
}

// IsAjax returns wheter the request was made via ajax. Internally,
// it uses X-Requested-With, which is set by all major JS libraries.
func (c *Context) IsAjax() bool {
	return c.R != nil && c.R.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

// Close closes any resources opened by the context.
// It's automatically called by the App, so you
// don't need to call it manually
func (c *Context) Close() {
	if c.app.debug {
		if o, ok := c.getDebug(debugOrmKey).(*Orm); ok {
			o.Close()
		}
		if ca, ok := c.getDebug(debugCacheKey).(*Cache); ok {
			ca.Close()
		}
	}
}

// BackgroundContext returns a copy of the given Context
// suitable for using after the request has been serviced
// (id est, in goroutines spawned from the Handler which
// might outlast the Handler's lifetime). Please, see also
// Finalize.
func (c *Context) BackgroundContext() *Context {
	ctx := c.app.NewContext(nil)
	ctx.R = c.R
	ctx.background = true
	ctx.provider = c.provider
	ctx.reProvider = c.reProvider
	ctx.ResponseWriter = discard
	return ctx
}

// Finalize recovers from any potential panics and then closes
// the Context. It is used in conjunction with BackgroundContext, to
// safely spawn background jobs from requests while also
// logging any potential errors. The common usage pattern is:
//
//  func MyHandler(ctx *app.Context) {
//	data := AcquireData(ctx)
//	c := ctx.BackgroundContext()
//	go func() {
//	    defer c.Finalize()
//	    CrunchData(c, data) // note the usage of c rather than ctx
//	}()
//	ctx.MustExecute("mytemplate.html", data)
//  }
func (c *Context) Finalize() {
	if err := recover(); err != nil {
		c.app.recoverErr(c, err)
	}
	c.app.CloseContext(c)
}

// Intercept http.ResponseWriter calls to find response
// status code

func (c *Context) WriteHeader(code int) {
	if c.statusCode < 0 {
		code = -c.statusCode
	}
	c.statusCode = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *Context) WriteString(s string) (int, error) {
	return c.Write([]byte(s))
}

func (c *Context) Write(data []byte) (int, error) {
	if c.statusCode == 0 {
		c.statusCode = http.StatusOK
	} else if c.statusCode < 0 {
		// code will be overriden
		c.WriteHeader(0)
	}
	return c.ResponseWriter.Write(data)
}

func urlHost(u string) string {
	if u, _ := url.Parse(u); u != nil {
		return u.Host
	}
	return ""
}
