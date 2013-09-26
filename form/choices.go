package form

import (
	"gnd.la/mux"
)

type Choice struct {
	Name  string
	Value interface{}
}

type ChoicesProvider interface {
	FieldChoices(ctx *mux.Context, field *Field) []*Choice
}
