package mux

import (
	"fmt"
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
	Name string
}

func (m *MissingParameterError) StatusCode() int {
	return http.StatusBadRequest
}

func (m *MissingParameterError) Error() string {
	return fmt.Sprintf("Missing required parameter \"%s\"", m.Name)
}

// Indicates that a parameter does not have the required type
// (e.g. an int was requested but a string was provided). Its
// status code is 400.
type InvalidParameterTypeError struct {
	*MissingParameterError
	Type reflect.Type
}

func (i *InvalidParameterTypeError) Error() string {
	return fmt.Sprintf("Required parameter %q must be of type %v", i.Name, i.Type)
}

// NotFound panics with NotFoundError.
func NotFound() {
	KindNotFound("")
}

// KindNotFound panics with a NotFoundError indicating
// what was not found (.e.g "Article not found).
func KindNotFound(kind string) {
	panic(&NotFoundError{kind})
}

// MissingParameter panics with a MissingParameterError, using
// the given parameter name.
func MissingParameter(name string) {
	panic(&MissingParameterError{name})
}

// InvalidParameterType panics with an InvalidParameterTypeError
// using the given parameter name and type.
func InvalidParameterType(name string, typ reflect.Type) {
	panic(&InvalidParameterTypeError{&MissingParameterError{name}, typ})
}
