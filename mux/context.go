package mux

import (
	"fmt"
	"gondola/cache"
	"gondola/errors"
	"math"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

type ContextFinalizer func(*Context)

type Context struct {
	http.ResponseWriter
	R             *http.Request
	submatches    []string
	params        map[string]string
	c             *cache.Cache
	cached        bool
	fromCache     bool
	handlerName   string
	mux           *Mux
	statusCode    int
	customContext interface{}
	Data          interface{} /* Left to the user */
}

// Count returns the number of elements captured
// by the pattern which matched the handler.
func (c *Context) Count() int {
	return len(c.submatches)
}

// IndexValue returns the captured parameter
// at the given index or an empty string if no
// such parameter exists.
func (c *Context) IndexValue(idx int) string {
	if idx < len(c.submatches) {
		return c.submatches[idx]
	}
	return ""
}

// ParseIndexValue uses the captured parameter
// at the given index and tries to parse it
// into the given argument. See ParseFormValue
// for examples as well as the supported types.
func (c *Context) ParseIndexValue(idx int, arg interface{}) bool {
	val := c.IndexValue(idx)
	return c.parseTypedValue(val, arg)
}

// ParamValue returns the named captured parameter
// with the given name or an empty string if it
// does not exist.
func (c *Context) ParamValue(name string) string {
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
	return strings.TrimSpace(c.R.FormValue(name))
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

// RequireParseFormValue works like ParseFormValue but raises a
// MissingParameterError if the parameter is missing or an
// InvalidParameterTypeError if the parameter does not have the
// required type
func (c *Context) RequireParseFormValue(name string, arg interface{}) {
	val := c.RequireFormValue(name)
	if !c.parseTypedValue(val, arg) {
		t := reflect.TypeOf(arg)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
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
// message and HTTP code.
func (c *Context) Error(error string, code int) {
	http.Error(c, error, code)
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

// C returns the custom type context wrapped in
// an interface{}. Intended for use in templates
// e.g. {{ Ctx.C.MyCustomMethod }}
//
// For use in code it's better to use the function
// you previously defined as the context transformation for
// the mux e.g.
// function in your own code to avoid type assertions
// func Context(ctx *mux.Context) *mycontext {
//	return (*mycontext)(ctx)
// }
// ...
// mymux.SetContextTransform(Context)
func (c *Context) C() interface{} {
	if c.customContext == nil {
		if c.mux.contextTransform != nil {
			result := c.mux.contextTransform.Call([]reflect.Value{reflect.ValueOf(c)})
			c.customContext = result[0].Interface()
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
}

// Intercept http.ResponseWriter calls to find response
// status code

func (c *Context) WriteHeader(code int) {
	c.ResponseWriter.WriteHeader(code)
}

func (c *Context) Write(data []byte) (int, error) {
	n, err := c.ResponseWriter.Write(data)
	if c.statusCode == 0 {
		c.statusCode = http.StatusOK
	}
	return n, err
}
