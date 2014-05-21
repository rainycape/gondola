package form

import (
	"fmt"

	"gnd.la/app"
	"gnd.la/i18n"
)

const (
	notChosen = "___gondola_not_chosen"
	// NotChosen is used to indicate that a value from
	// a multiple choices input has not been chosen
	// by the user. See also Choose.
	NotChosen = notChosen
)

var (
	// Choose creates a choice entry with the text
	// "Please, choose" which generates an error when
	// the user does not choose anything.
	Choose = &Choice{
		Name:  i18n.String("form|Please, choose"),
		Value: NotChosen,
	}
)

// Choice represents a choice in a select or radio field.
type Choice struct {
	// Name must be either a string or an i18n.String
	// or a fmt.Stringer.
	// Other types will panic when rendering the form.
	Name  interface{}
	Value interface{}
}

func (c *Choice) TranslatedName(lang i18n.Languager) string {
	switch x := c.Name.(type) {
	case string:
		return x
	case i18n.String:
		return x.TranslatedString(lang)
	case fmt.Stringer:
		return x.String()
	}
	panic(fmt.Errorf("choice %+v has invalid Name type %T", c, c.Name))
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
