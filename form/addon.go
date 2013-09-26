package form

import (
	"gnd.la/html"
	"gnd.la/mux"
)

type AddOnPosition int

const (
	AddOnPositionBefore = iota
	AddOnPositionAfter
)

type AddOn struct {
	Node     *html.Node
	Position AddOnPosition
}

type AddOnProvider interface {
	FieldAddOns(ctx *mux.Context, field *Field) []*AddOn
}
