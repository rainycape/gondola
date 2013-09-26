package form

import (
	"gnd.la/html"
	"gnd.la/mux"
)

// AddOnPosition indicates the position of the addon
// relative to the field input.
type AddOnPosition int

const (
	// Before the input
	AddOnPositionBefore = iota
	// After the input
	AddOnPositionAfter
)

// AddOn represents an addon which can be added
// to a form field. Keep in mind that not all
// renderers nor all types of fields support
// addons. Check each renderer's documentation
// for further information.
type AddOn struct {
	// The HTML node to use as addon
	Node *html.Node
	// The addon position
	Position AddOnPosition
}

// AddOnProvider is an interface which might be implemented
// by types included in a form. If the type implements this
// interface, its function will be called for every field.
type AddOnProvider interface {
	// FieldAddOns returns the addons for the given field.
	// It's called just before the field is rendered.
	FieldAddOns(ctx *mux.Context, field *Field) []*AddOn
}
