package mux

import (
	"fmt"
	"gondola/cache"
	"gondola/cookies"
	"gondola/defaults"
	"gondola/errors"
	"gondola/log"
	"gondola/orm"
	"gondola/serialize"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type ContextFinalizer func(*Context)

type Context struct {
	http.ResponseWriter
	R             *http.Request
	arguments     []string
	params        map[string]string
	re            *regexp.Regexp
	c             *cache.Cache
	cached        bool
	fromCache     bool
	handlerName   string
	mux           *Mux
	statusCode    int
	customContext interface{}
	started       time.Time
	cookies       *cookies.Cookies
	o             *orm.Orm
	Data          interface{} /* Left to the user */
}

func (c *Context) reset(argcap int) {
	c.ResponseWriter = nil
	c.R = nil
	if cap(c.arguments) == argcap {
		c.arguments = c.arguments[:0]
	} else {
		c.arguments = make([]string, 0, argcap)
	}
	c.params = nil
	c.re = nil
	c.cached = false
	c.fromCache = false
	c.statusCode = 0
	c.customContext = nil
	c.started = time.Now()
	c.cookies = nil
	c.Data = nil
}

// Count returns the number of elements captured
// by the pattern which matched the handler.
func (c *Context) Count() int {
	return len(c.arguments) - 1
}

// IndexValue returns the captured parameter
// at the given index or an empty string if no
// such parameter exists. Pass -1 to obtain
// the whole match.
func (c *Context) IndexValue(idx int) string {
	if idx >= -1 && idx < len(c.arguments)-1 {
		return c.arguments[idx+1]
	}
	return ""
}

// RequireIndexValue works like IndexValue, but raises
// a MissingParameter error if the value is not present
// or empty.
func (c *Context) RequireIndexValue(idx int) string {
	val := c.IndexValue(idx)
	if val == "" {
		errors.MissingParameter(fmt.Sprintf("at index %d", idx))
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
	if c.params == nil {
		if c.re == nil {
			return ""
		}
		params := map[string]string{}
		for ii, n := range c.re.SubexpNames() {
			if ii == len(c.arguments) {
				break
			}
			if n != "" {
				params[n] = c.arguments[ii]
			}
		}
		c.params = params
	}
	return c.params[name]
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
		errors.MissingParameter(name)
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
		errors.InvalidParameterType(name, t.String())
	}
}

// StatusCode returns the response status code. If the headers
// haven't been written yet, it returns 0
func (c *Context) StatusCode() int {
	return c.statusCode
}

func (c *Context) funcName(depth int) string {
	funcName := "???"
	caller, _, _, ok := runtime.Caller(depth + 1)
	if ok {
		f := runtime.FuncForPC(caller)
		if f != nil {
			fullName := strings.Trim(f.Name(), ".")
			parts := strings.Split(fullName, ".")
			simpleName := parts[len(parts)-1]
			funcName = fmt.Sprintf("%s()", simpleName)
		}
	}
	return funcName
}

func (c *Context) parseTypedValue(val string, arg interface{}) bool {
	v := reflect.ValueOf(arg)
	for v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.CanSet() {
		panic(fmt.Errorf("Invalid argument type passed to %s. Please pass %s instead of %s.",
			c.funcName(1), reflect.PtrTo(v.Type()), v.Type()))
	}
	switch v.Type().Kind() {
	case reflect.Bool:
		res := false
		if val != "" && val != "0" && strings.ToLower(val) != "false" {
			res = true
		}
		v.SetBool(res)
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		res, err := strconv.ParseInt(val, 0, 64)
		if err == nil {
			if v.OverflowInt(res) {
				if res > 0 {
					res = int64(math.Pow(2, float64(8*v.Type().Size()-1)) - 1)
				} else {
					res = -int64(math.Pow(2, float64(8*v.Type().Size()-1)))
				}
			}
			v.SetInt(res)
			return true
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		res, err := strconv.ParseUint(val, 0, 64)
		if err == nil {
			if v.OverflowUint(res) {
				res = uint64(math.Pow(2, float64(8*v.Type().Size())) - 1)
			}
			v.SetUint(res)
			return true
		}
	case reflect.Float32, reflect.Float64:
		res, err := strconv.ParseFloat(val, 64)
		if err == nil {
			v.SetFloat(res)
			return true
		}
	case reflect.String:
		v.SetString(val)
		return true
	default:
		panic(fmt.Errorf("Invalid arguent type passed to %s: %s. Please, see ParseFormValue() docs for a list of the supported types.",
			c.funcName(1), v.Type()))
	}
	return false
}

// Cache returns the default cache
// See gondola/cache for a further
// explanation
func (c *Context) Cache() *cache.Cache {
	if c.c == nil {
		c.c = cache.NewDefault()
	}
	return c.c
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
// has been defined for the mux, it will be
// given the opportunity to intercept the
// error and provide its own response.
func (c *Context) Error(error string, code int) {
	c.statusCode = -code
	c.mux.handleHTTPError(c, error, code)
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
// (see gondola/cache/layer)
func (c *Context) Cached() bool {
	return c.cached
}

// ServedFromCache returns true if the request
// was served by a cache layer
// (see gondola/cache/layer)
func (c *Context) ServedFromCache() bool {
	return c.fromCache
}

// HandlerName returns the name of the handler which
// handled this context
func (c *Context) HandlerName() string {
	return c.handlerName
}

// Mux returns the Mux this Context originated from
func (c *Context) Mux() *Mux {
	return c.mux
}

// MustReverse calls MustReverse on the mux this context originated
// from. See the documentation on Mux for details.
func (c *Context) MustReverse(name string, args ...interface{}) string {
	return c.mux.MustReverse(name, args...)
}

// Reverse calls Reverse on the mux this context originated
// from. See the documentation on Mux for details.
func (c *Context) Reverse(name string, args ...interface{}) (string, error) {
	return c.mux.Reverse(name, args...)
}

// RedirectReverse calls Reverse to find the URL and then sends
// the redirect to the client. See the documentation on Mux.Reverse
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

// Cookies returns a coookies.Cookies object which
// can be used to set and delete cookies. See the documentation
// on gondola/cookies for more information.
func (c *Context) Cookies() *cookies.Cookies {
	if c.cookies == nil {
		mux := c.Mux()
		c.cookies = cookies.New(c.R, c, mux.Secret(),
			mux.EncryptionKey(), mux.DefaultCookieOptions())
	}
	return c.cookies
}

// Orm returns a connection to the ORM using the default database
// and panics if there's an error. See the the documentation on
// gondola/defaults/SetDatabase for further information.
func (c *Context) Orm() *orm.Orm {
	if c.o == nil {
		driver, source := defaults.DatabaseParameters()
		if driver == "" {
			panic(fmt.Errorf("Default database is not set"))
		}
		log.Debugf("Opening ORM connection %s:%s", driver, source)
		var err error
		c.o, err = orm.Open(driver, source)
		if err != nil {
			panic(err)
		}
	}
	return c.o
}

// Execute loads the template with the given name using the
// mux template loader and executes it with the data argument.
func (c *Context) Execute(name string, data interface{}) error {
	tmpl, err := c.mux.LoadTemplate(name)
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

// WriteJson is equivalent to serialize.WriteJson(ctx, data)
func (c *Context) WriteJson(data interface{}) (int, error) {
	return serialize.WriteJson(c, data)
}

// WriteXml is equivalent to serialize.WriteXml(ctx, data)
func (c *Context) WriteXml(data interface{}) (int, error) {
	return serialize.WriteXml(c, data)
}

// Custom returns the custom type context wrapped in
// an interface{}. It's intended for use in templates
// e.g. {{ $Context.Custom.MyCustomMethod }}
//
// For use in code it's better to use the a conveniency function
// to transform the type without any type assertions e.g.
//
//	type mycontext mux.Context
//	func Context(ctx *mux.Context) *mycontext {
//	    return (*mycontext)(ctx)
//	}
//	mymux.SetCustomContextType(&mycontext{})
func (c *Context) Custom() interface{} {
	if c.customContext == nil {
		if c.mux.customContextType != nil {
			c.customContext = reflect.ValueOf(c).Convert(*c.mux.customContextType).Interface()
		} else {
			c.customContext = c
		}
	}
	return c.customContext
}

// Close closes any resources opened by the context
// (for now, the cache connection). It's automatically
// called by the mux, so you don't need to call it
// manually
func (c *Context) Close() {
	if c.c != nil {
		c.c.Close()
		c.c = nil
	}
	if c.o != nil {
		log.Debug("Closing ORM connection")
		c.o.Close()
		c.o = nil
	}
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

func (c *Context) Write(data []byte) (int, error) {
	if c.statusCode == 0 {
		c.statusCode = http.StatusOK
	} else if c.statusCode < 0 {
		// code will be overriden
		c.WriteHeader(0)
	}
	return c.ResponseWriter.Write(data)
}
