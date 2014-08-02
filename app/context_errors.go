package app

import (
	"fmt"
	"net/http"

	"gnd.la/i18n"
)

var (
	defaultMessages = map[int]i18n.String{
		http.StatusBadRequest:                   i18n.String("bad request"),
		http.StatusUnauthorized:                 i18n.String("unauthorized"),
		http.StatusPaymentRequired:              i18n.String("payment required"),
		http.StatusForbidden:                    i18n.String("forbidden"),
		http.StatusNotFound:                     i18n.String("not found"),
		http.StatusMethodNotAllowed:             i18n.String("method not allowed"),
		http.StatusNotAcceptable:                i18n.String("not acceptable"),
		http.StatusProxyAuthRequired:            i18n.String("proxy required"),
		http.StatusRequestTimeout:               i18n.String("request timeout"),
		http.StatusConflict:                     i18n.String("conflict"),
		http.StatusGone:                         i18n.String("gone"),
		http.StatusLengthRequired:               i18n.String("length required"),
		http.StatusPreconditionFailed:           i18n.String("precondition failed"),
		http.StatusRequestEntityTooLarge:        i18n.String("request entity too large"),
		http.StatusRequestURITooLong:            i18n.String("request URI too long"),
		http.StatusUnsupportedMediaType:         i18n.String("unsupported media type"),
		http.StatusRequestedRangeNotSatisfiable: i18n.String("request range not satisfiable"),
		http.StatusExpectationFailed:            i18n.String("expectation failed"),
		http.StatusTeapot:                       i18n.String("i'm a teapot"),

		http.StatusInternalServerError:     i18n.String("internal server error"),
		http.StatusNotImplemented:          i18n.String("status not implemented"),
		http.StatusBadGateway:              i18n.String("bad gateway"),
		http.StatusServiceUnavailable:      i18n.String("service unavailable"),
		http.StatusGatewayTimeout:          i18n.String("gateway timeout"),
		http.StatusHTTPVersionNotSupported: i18n.String("HTTP version not supported"),
	}
)

func (c *Context) error(code int, message string) {
	c.statusCode = -code
	c.app.handleHTTPError(c, message, code)
}

// Error replies to the request with the specified HTTP code.
// The error message is constructed from the given arguments
// using fmt.Sprint. If an error handler has been defined for the App
// (see App.SetErrorHandler), it will be given the opportunity to intercept
// the error and provide its own response.
//
// Note that the standard 4xx and 5xx errors will use a default error
// message if none is provided.
//
// See also Context.Errorf, Context.NotFound, Context.NotFoundf,
// Context.Forbidden, Context.Forbiddenf, Context.BadRequest
// and Context.BadRequestf.
func (c *Context) Error(code int, args ...interface{}) {
	message := fmt.Sprint(args...)
	if message == "" {
		message = defaultMessages[code].TranslatedString(c)
	}
	c.error(code, message)
}

// Errorf works like Context.Error, but formats the error message
// using fmt.Sprintf.
func (c *Context) Errorf(code int, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	c.error(code, message)
}

// NotFound is equivalent to Context.Error() with http.StatusNotFound
// as the error code.
func (c *Context) NotFound(args ...interface{}) {
	c.Error(http.StatusNotFound, args...)
}

// NotFoundf is equivalent to Context.Errorf() with http.StatusNotFound
// as the error code.
func (c *Context) NotFoundf(format string, args ...interface{}) {
	c.Errorf(http.StatusNotFound, format, args...)
}

// Forbidden is equivalent to to Context.Error() with http.StatusForbidden
// as the error code.
func (c *Context) Forbidden(args ...interface{}) {
	c.Error(http.StatusForbidden, args...)
}

// Forbiddenf is equivalent to to Context.Errorf() with http.StatusForbidden
// as the error code.
func (c *Context) Forbiddenf(format string, args ...interface{}) {
	c.Errorf(http.StatusForbidden, format, args...)
}

// BadRequest is equivalent to calling Context.Error() with http.StatusBadRequest
// as the error code.
func (c *Context) BadRequest(args ...interface{}) {
	c.Error(http.StatusBadRequest, args...)
}

// BadRequestf is equivalent to calling Context.Errorf() with http.StatusBadRequest
// as the error code.
func (c *Context) BadRequestf(format string, args ...interface{}) {
	c.Errorf(http.StatusBadRequest, format, args...)
}
