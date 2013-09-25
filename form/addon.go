package form

import (
	"gnd.la/html"
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
	FieldAddOns(field *Field) []*AddOn
}
