package app

import (
	"fmt"
	"net"
	"net/http"
	"reflect"
)

// Error represents an error which, besides a message,
// has also an assocciated status code to be sent to
// the client.
type Error interface {
	error
	StatusCode() int
}

// Indicates that something wasn't found. Its status code
// is 404.
type NotFoundError struct {
	Kind string
}

func (n *NotFoundError) StatusCode() int {
	return http.StatusNotFound
}

func (n *NotFoundError) Error() string {
	if n.Kind != "" {
		fmt.Sprintf("%s not found", n.Kind)
	}
	return "Not found"
}

// Indicates that a required parameter is mising. Its
// status code is 400.
type MissingParameterError struct {
	// The parameter index if it was requested
	// via its index (e.g. ParseIndexValue)
	Index int
	// The parameter name if it was requested via
	// form name or parameter name (e.g. ParseFormValue
	// or ParseParamValue).
	Name string
}

func (e *MissingParameterError) StatusCode() int {
	return http.StatusBadRequest
}

func (e *MissingParameterError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("missing required parameter %q", e.Name)
	}
	return fmt.Sprintf("missing required parameter at index %d", e.Index)
}

// Indicates that a parameter does not have the required type
// (e.g. an int was requested but a string was provided). Its
// status code is 400.
type InvalidParameterError struct {
	// The parameter index if it was requested
	// via its index (e.g. ParseIndexValue)
	Index int
	// The parameter name if it was requested via
	// form name or parameter name (e.g. ParseFormValue
	// or ParseParamValue).
	Name string
	// The type expected to be able to parse the value
	// into.
	Type reflect.Type
	// The underlying error. Might be nil.
	Err error
}

func (e *InvalidParameterError) StatusCode() int {
	return http.StatusBadRequest
}

func (e *InvalidParameterError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("can't parse parameter %q as %s", e.Name, e.Type)
	}
	return fmt.Sprintf("can't parse parameter at index %d as %s", e.Index, e.Type)
}

func isIgnorable(err interface{}) bool {
	if e, ok := err.(error); ok {
		if ne, ok := e.(*net.OpError); ok {
			e = ne.Err
		}
		if e == ePIPE || e == eCONNRESET {
			// Client closed the connection
			return true
		}
	}
	return false
}
