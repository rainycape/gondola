package errors

import (
	"fmt"
	"net/http"
)

type Error interface {
	String() string
	StatusCode() int
}

type NotFoundError struct {
	Kind string
}

func (n *NotFoundError) StatusCode() int {
	return http.StatusNotFound
}

func (n *NotFoundError) String() string {
	if n.Kind != "" {
		fmt.Sprintf("%s not found", n.Kind)
	}
	return "Not found"
}

type MissingParameterError struct {
	ParameterName string
}

func (m *MissingParameterError) StatusCode() int {
	return http.StatusBadRequest
}

func (m *MissingParameterError) String() string {
	return fmt.Sprintf("Missing required parameter \"%s\"", m.ParameterName)
}

type InvalidParameterTypeError struct {
    *MissingParameterError
    ParameterType string
}

func (i *InvalidParameterTypeError) String() string {
    return fmt.Sprintf("Required parameter \"%s\" must be of type %s", i.ParameterName, i.ParameterType)
}
