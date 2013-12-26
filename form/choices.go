package form

import (
	"gnd.la/app"
)

// Choice represents a choice in a select or radio field.
type Choice struct {
	Name  string
	Value interface{}
}

// ChoicesProvider is the interface implemented by types
// which contain a field or type select or radio. The
// function will only be called for fields of those types.
// If a type which contains choices does not implement this
// interface, the form will panic on creation.
type ChoicesProvider interface {
	// FieldChoices returns the choices for the given field.
	// It's called just before the field is rendered.
	FieldChoices(ctx *app.Context, field *Field) []*Choice
}
