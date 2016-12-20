package app

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	"gnd.la/app/cookies"
	"gnd.la/app/profile"
	"gnd.la/app/serialize"
	"gnd.la/blobstore"
	"gnd.la/cache"
	"gnd.la/form/input"
	"gnd.la/i18n/table"
	"gnd.la/internal"
	"gnd.la/kvs"
	"gnd.la/log"
	"gnd.la/net/urlutil"
	"gnd.la/orm"
	"gnd.la/util/types"
)

var (
	// CookieSalt is the default salt used for signing
	// cookies. For extra security, you might change this
	// value but keep in mind that all previously issued
	// signed cookies will be invalidated, since their
	// signature won't match.
	CookieSalt = []byte("gnd.la/app/cookies.salt")
)

type ContextFinalizer func(*Context)

type Context struct {
	http.ResponseWriter
	R               *http.Request
	bodyReader      io.Reader
	provider        ContextProvider
	reProvider      *regexpProvider
	handlerName     string
	app             *App
	statusCode      int
	started         time.Time
	cookies         *cookies.Cookies
	user            User
	translations    *table.Table
	hasTranslations bool
	background      bool
	wg              *sync.WaitGroup
	kv              kvs.KVS
}

func (c *Context) reset() {
	c.ResponseWriter = nil
	c.R = nil
	c.bodyReader = nil
	c.statusCode = 0
	c.started = time.Now()
	c.cookies = nil
	c.user = nil
	c.translations = nil
	c.hasTranslations = false
	c.kv.Clear()
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
	val, _ := c.provider.Arg(idx)
	return val
}

// RequireIndexValue works like IndexValue, but panics
// with a MissingParameter error if the value is not present
// or empty.
func (c *Context) RequireIndexValue(idx int) string {
	val := c.IndexValue(idx)
	if val == "" {
		panic(&MissingParameterError{Index: idx})
	}
	return val
}

// ParseIndexValue uses the captured parameter
// at the given index and tries to parse it
// into the given argument. See Context.ParseFormValue
// for more details.
func (c *Context) ParseIndexValue(idx int, arg interface{}) error {
	val, found := c.provider.Arg(idx)
	return c.parseTypedValue(idx, "", val, found, arg)
}

// MustParseIndexValue works like ParseIndexValue but panics in
// case of error.
func (c *Context) MustParseIndexValue(idx int, arg interface{}) {
	if err := c.ParseIndexValue(idx, arg); err != nil {
		panic(err)
	}
}

// ParamValue returns the named captured parameter
// with the given name or an empty string if it
// does not exist.
func (c *Context) ParamValue(name string) string {
	val, _ := c.provider.Param(name)
	return val
}

// ParseParamValue uses the named captured parameter
// with the given name and tries to parse it into
// the given argument. See Context.ParseFormValue
// for more details.
func (c *Context) ParseParamValue(name string, arg interface{}) error {
	val, found := c.provider.Param(name)
	return c.parseTypedValue(-1, name, val, found, arg)
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

// RequireFormValue works like FormValue, but panics with
// a MissingParameter error if the value is not present
// or empty.
func (c *Context) RequireFormValue(name string) string {
	val := c.FormValue(name)
	if val == "" {
		panic(&MissingParameterError{Name: name})
	}
	return val
}

// ParseFormValue tries to parse the named form value into the given
// arg. If the value is not present, a MissingParameterError is
// returned. If the value is present but it fails to parse, it
// returns an InvalidParameterError.
//
// Supported types are: bool, u?int(8|16|32|64)? and float(32|64)
// Internally, ParseFormValue uses gnd.la/form/input to parse
// its arguments, so any type implementing gnd.la/form/input.Parser
// can be used too.
//
// Note that arg is not altered the returned error is non-nil.
//
// Some examples (with error handling omitted):
//
//  var f float32
//  ctx.ParseFormValue("quality", &f)
//
//  var width uint
//  ctx.ParseFormValue("width", &width)
func (c *Context) ParseFormValue(name string, arg interface{}) error {
	val := c.FormValue(name)
	found := true
	if val == "" && c.R != nil {
		// Check if the value was really provided
		if vs := c.R.Form[name]; len(vs) == 0 {
			// Parameter not provided
			found = false
		}
	}
	return c.parseTypedValue(-1, name, val, found, arg)
}

// MustParseFormValue works like ParseFormValue but panics if
// there's an error.
func (c *Context) MustParseFormValue(name string, arg interface{}) {
	if err := c.ParseFormValue(name, arg); err != nil {
		panic(err)
	}
}

// Provider returns the Provider which the Context uses
// to retrieve its arguments.
func (c *Context) Provider() ContextProvider {
	return c.provider
}

// StatusCode returns the response status code. If no status code
// has been set yet, it returns the OK code (200).
func (c *Context) StatusCode() int {
	if c.statusCode < 0 {
		return -c.statusCode
	}
	if c.statusCode == 0 {
		return http.StatusOK
	}
	return c.statusCode
}

func (c *Context) SetStatusCode(code int) {
	c.statusCode = -code
}

func (c *Context) parseTypedValue(idx int, name string, val string, found bool, arg interface{}) error {
	if !found {
		return &MissingParameterError{Index: idx, Name: name}
	}
	if err := input.Parse(val, arg); err != nil {
		if err == types.ErrCantSet {
			// Programming error, user did not pass a pointer
			panic(err)
		}
		return &InvalidParameterError{
			Index: idx,
			Name:  name,
			Type:  reflect.TypeOf(arg),
			Err:   err,
		}
	}
	return nil
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

func (c *Context) boolValue(key string) bool {
	if val, ok := c.kv.Get(key).(bool); ok {
		return val
	}
	return false
}

// Cached returns true if the request was cached by a cache layer.
// See gnd.la/cache/layer for more information.
func (c *Context) Cached() bool {
	return c.boolValue(internal.LayerCachedKey)
}

// ServedFromCache returns true if the request was served by a cache layer.
// See gnd.la/cache/layer for more information.
func (c *Context) ServedFromCache() bool {
	return c.boolValue(internal.LayerServedFromCacheKey)
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

// Request returns the *http.Request associated with this
// context. Note that users should access the Context.R
// field directly, rather than using this method (it solely
// exists for App Engine compatibility).
func (c *Context) Request() *http.Request {
	return c.R
}

// MustReverse is a shorthand for Context.Reverse which panics instead
// of returning an error.
func (c *Context) MustReverse(name string, args ...interface{}) string {
	r, err := c.Reverse(name, args...)
	if err != nil {
		panic(err)
	}
	return r
}

// Reverse calls Reverse on the App this context originated from. See
// App.Reverse for details. Note that Context.Reverse might return a different
// value than App.Reverse for host-specific handlers, since App.Reverse will
// return a protocol-relative URL (e.g. //www.gondolaweb.com) while Context.Reverse
// can return an absolute URL (e.g. http://www.gondolaweb.com) if the Context
// has a Request associated with it.
func (c *Context) Reverse(name string, args ...interface{}) (string, error) {
	r, err := c.app.Reverse(name, args...)
	if err == nil && strings.HasPrefix(r, "//") {
		if s := c.requestScheme(); s != "" {
			r = s + ":" + r
		}
	}
	return r, err
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
		if from != "" && urlutil.SameHost(from, us) {
			redir = from
		} else if ref := c.R.Referer(); ref != "" && urlutil.SameHost(ref, us) {
			redir = ref
		}
		// us can be protocol relative
		if us == redir || strings.HasPrefix(us, "//") && c.R.Host == urlHost(redir) {
			redir = "/"
		}
		c.Redirect(redir, false)
	}
}

func (c *Context) requestScheme() string {
	if c.R != nil {
		if c.R.TLS != nil {
			return "https"
		}
		return "http"
	}
	return ""
}

// URL return the absolute URL for the current request. Note that
// if your app is running behind a proxy, you might need to properly
// configure the App as well as the appropriate X-headers in your
// proxy and your app (X-Forwarded-For, X-Scheme, etc...).
func (c *Context) URL() *url.URL {
	if c.R != nil {
		u := *c.R.URL
		u.Host = c.R.Host
		if u.Scheme == "" {
			u.Scheme = c.requestScheme()
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
		signer, _ := c.app.Signer(CookieSalt)
		encrypter, _ := c.app.Encrypter()
		c.cookies = cookies.New(c.R, c, c.app.CookieCodec, signer,
			encrypter, c.app.CookieOptions)
	}
	return c.cookies
}

// Cache is a shorthand for ctx.App().Cache(), but panics in case
// of error, instead of returning it.
func (c *Context) Cache() *cache.Cache {
	return c.cache()
}

// Blobstore is a shorthand for ctx.App().Blobstore(), but panics in
// case of error instead of returning it.
func (c *Context) Blobstore() *blobstore.Blobstore {
	return c.blobstore()
}

// Orm is a shorthand for ctx.App().Orm(), but panics in case
// of error, rather than returning it.
func (c *Context) Orm() *orm.Orm {
	return c.orm()
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

// DecodeJSON creates a json.Decoder reading from
// the Context and calls Decode with the v argument.
// See also Context.Read for more information on
// transparent decompression.
func (c *Context) DecodeJSON(v interface{}) error {
	dec := json.NewDecoder(c)
	return dec.Decode(v)
}

// MustDecodeJSON is a shorthand for DecodeJSON which panics
// instead of returning an error.
func (c *Context) MustDecodeJSON(v interface{}) {
	if err := c.DecodeJSON(v); err != nil {
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

// GetHeader is a shorthand for Context.R.Header.Get()
func (c *Context) GetHeader(key string) string {
	if c != nil && c.R != nil {
		return c.R.Header.Get(key)
	}
	return ""
}

// SetHeader is a shorthand for Context.Header().Set
func (c *Context) SetHeader(key string, value string) {
	h := c.Header()
	if h != nil {
		h.Set(key, value)
	}
}

// AddHeader is a shorthand for Context.Header().Add
func (c *Context) AddHeader(key string, value string) {
	h := c.Header()
	if h != nil {
		h.Add(key, value)
	}
}

// IsXHR returns wheter the request was made via XMLHTTPRequest. Internally,
// it uses X-Requested-With, which is set by all major JS libraries.
func (c *Context) IsXHR() bool {
	return c.GetHeader("X-Requested-With") == "XMLHttpRequest"
}

// Close closes any resources opened by the context.
// It's automatically called by the App, so you
// don't need to call it manually
func (c *Context) Close() {
	// currently a no-op
}

// BackgroundContext returns a copy of the given Context
// suitable for using after the request has been serviced
// (id est, in goroutines spawned from the Handler which
// might outlast the Handler's lifetime). Please, see also
// Finalize and Go.
func (c *Context) backgroundContext() *Context {
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
//
// See also Go.
func (c *Context) finalize(wg *sync.WaitGroup) {
	wg.Done()
	if err := recover(); err != nil {
		c.app.recoverErr(c, err)
	}
	c.app.CloseContext(c)
}

// Go spawns a new goroutine using a copy of the given Context
// suitable for using after the request has been serviced
// (id est, in goroutines spawned from the Handler which
// might outlast the Handler's lifetime). Additionaly, Go also
// handles error recovering and profiling in the spawned
// goroutine. The initial Context can also wait for all
// background contexts to finish by calling Wait().
//
// In the following example, the handler finishes and returns the
// executed template while CrunchData is still potentially running.
//
//  func MyHandler(ctx *app.Context) {
//	data := AcquireData(ctx)
//	ctx.Go(func (c *app.Context) {
//	    CrunchData(c, data) // note the usage of c rather than ctx
//	}
//	ctx.MustExecute("mytemplate.html", data)
//  }
func (c *Context) Go(f func(*Context)) {
	if c.wg == nil {
		c.wg = new(sync.WaitGroup)
	}
	c.wg.Add(1)
	bg := c.backgroundContext()
	var id int
	if profile.On {
		id = profile.ID()
	}
	go func() {
		if profile.On {
			profile.Begin()
			defer profile.End(id)
		}
		defer bg.finalize(c.wg)
		f(bg)
	}()
}

// Wait waits for any pending background contexts spawned from
// Go() to finish. If there are no background contexts spawned
// from this context, this functions returns immediately.
func (c *Context) Wait() {
	if c.wg != nil {
		c.wg.Wait()
	}
}

// Logger returns a Logger which allows logging mesages in several
// levels. See gnd.la/log.Interface interface for more information.
// Note that this function will always return non-nil even when logging
// is disabled, so it's safe to call any gnd.la/log.Interface methods
// unconditionally (i.e. don't check if the returned value is nil, it'll
// never be).
func (c *Context) Logger() log.Interface {
	return c.logger()
}

// Read implements the io.Reader interface, reading from the request
// body. If there's no available request (e.g. the context originated
// from a scheduled task), Read() will always return 0, io.EOF. If
// the request headers indicate that the body is compressed (via
// Content-Encoding header), Read will automatically decompress the data
// for "gzip" and "deflate". Note that request headers won't be altered
// and callers can always read the raw request body data using Context.R.Body.
func (c *Context) Read(p []byte) (n int, err error) {
	if c.R == nil || c.R.Body == nil {
		return 0, io.EOF
	}
	if c.bodyReader == nil {
		switch c.R.Header.Get("Content-Encoding") {
		case "gzip":
			gzr, err := gzip.NewReader(c.R.Body)
			if err != nil {
				return 0, err
			}
			c.bodyReader = gzr
		case "deflate":
			zr, err := zlib.NewReader(c.R.Body)
			if err != nil {
				return 0, err
			}
			c.bodyReader = zr
		default:
			c.bodyReader = c.R.Body
		}
	}
	return c.bodyReader.Read(p)
}

// Intercept http.ResponseWriter calls to find response
// status code

func (c *Context) WriteHeader(code int) {
	if c.statusCode < 0 {
		code = -c.statusCode
	}
	c.statusCode = code
	if profile.On && profile.Profiling() {
		header := profileHeader(c)
		c.Header().Set(profile.HeaderName, header)
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *Context) WriteString(s string) (int, error) {
	return c.Write([]byte(s))
}

func (c *Context) Write(data []byte) (int, error) {
	if c.statusCode <= 0 {
		// code will be overriden if < 0
		c.WriteHeader(http.StatusOK)
	}
	return c.ResponseWriter.Write(data)
}

func urlHost(u string) string {
	if u, _ := url.Parse(u); u != nil {
		return u.Host
	}
	return ""
}
