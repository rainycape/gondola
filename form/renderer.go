package form

import (
	"gnd.la/html"
	"io"
)

// Renderer is the interface implemented to help the form
// render its elements. Without a Renderer, the form won't
// render errors nor help messages. Each function is called
// at most once for each non-hidden field included in the form.
type Renderer interface {
	// BeginField is called before starting to write any field.
	BeginField(w io.Writer, field *Field) error
	// LabelAttributes is called just before writing the <label> for the field.
	LabelAttributes(field *Field) (html.Attrs, error)
	// BeginInput is called just before writing the <input> or equivalent tag or any addons
	BeginInput(w io.Writer, field *Field) error
	// FieldAttributes is called just before writing the field (input, textarea, etc...)
	FieldAttributes(field *Field) (html.Attrs, error)
	// EndInput is called just after writing the <input> or equivalent tag and all the addons
	EndInput(w io.Writer, field *Field) error
	// WriteAddOn might be called multiple times, both before writing the field
	// and after (depending on the addons' positions).
	WriteAddOn(w io.Writer, field *Field, addon *AddOn) error
	// WriteError is called only for fields which are in not valid, after
	// the label and the input have been written.
	WriteError(w io.Writer, field *Field, err error) error
	// WriteHelp is called only for fields which have declared a help string, after
	// the label, the input and potentially the error have been written.
	WriteHelp(w io.Writer, field *Field) error
	// EndField is called after all the other field related functions have been called.
	EndField(w io.Writer, field *Field) error
}
