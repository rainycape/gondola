package form

import (
	"gnd.la/html"
	"io"
)

// Renderer is the interface implemented to help the form
// render its elements. Without a Renderer, the form won't
// render errors nor help messages. Except when noted otherwise,
// each function is called at most once for each non-hidden field
// included in the form.
type Renderer interface {
	// BeginField is called before starting to write any field.
	BeginField(w io.Writer, field *Field) error
	// BeginLabel is called before writing the label. It might be
	// called multiple times for radio fields.
	BeginLabel(w io.Writer, field *Field, pos int) error
	// LabelAttributes is called just before writing the <label> for the field.
	LabelAttributes(field *Field, pos int) (html.Attrs, error)
	// EndLabel is called before writing the end of the label. It might be
	// called multiple times for radio fields.
	EndLabel(w io.Writer, field *Field, pos int) error
	// BeginInput is called just before writing the <input> or equivalent tag
	// or any addons. For radio fields BeginInput will be called multiple times,
	// one per option.
	BeginInput(w io.Writer, field *Field, pos int) error
	// FieldAttributes is called just before writing the field (input, textarea, etc...)
	FieldAttributes(field *Field, pos int) (html.Attrs, error)
	// EndInput is called just after writing the <input> or equivalent tag and
	// all the addons. For radio fields EndInput will be called multiple times,
	// one per option.
	EndInput(w io.Writer, field *Field, pos int) error
	// WriteAddOn might be called multiple times, both before writing the field
	// and after (depending on the addons' positions). All these calls will happen
	// after BeginInput() and EndInput().
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
