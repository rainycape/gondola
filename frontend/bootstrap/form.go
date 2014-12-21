package bootstrap

import (
	"bytes"
	"fmt"
	"io"

	"gnd.la/form"
	"gnd.la/html"
)

const (
	numCols = 12
)

// FormRenderer implements a gnd.la/form renderer using
// bootstrap.
type FormRenderer struct {
	inputCols map[Size]int
}

// SetInputColumns sets the number of grid columns used for the input
// fields with the given size. This is frequently used in conjunction
// with .form-horizontal.
func (r *FormRenderer) SetInputColumns(size Size, columns int) {
	if r.inputCols == nil {
		r.inputCols = make(map[Size]int)
	}
	r.inputCols[size] = columns
}

// InputColumns returns the number of input columns for the given
// ize. See also SetInputColumns.
func (r *FormRenderer) InputColumns(size Size) int {
	return r.inputCols[size]
}

func (r *FormRenderer) inlineLabelClass() string {
	if len(r.inputCols) > 0 {
		var buf bytes.Buffer
		for k, v := range r.inputCols {
			fmt.Fprintf(&buf, "col-%s-%d col-%s-offset-%d", k.String(), v, k.String(), numCols-v)
		}
		return buf.String()
	}
	return ""
}

func (r *FormRenderer) labelClass() string {
	if len(r.inputCols) > 0 {
		var buf bytes.Buffer
		for k, v := range r.inputCols {
			fmt.Fprintf(&buf, "col-%s-%d ", k.String(), numCols-v)
		}
		return buf.String()
	}
	return ""
}

func (r *FormRenderer) inputDivClass() string {
	if len(r.inputCols) > 0 {
		var buf bytes.Buffer
		for k, v := range r.inputCols {
			fmt.Fprintf(&buf, "col-%s-%d ", k.String(), v)
		}
		return buf.String()
	}
	return ""
}

func (r *FormRenderer) BeginField(w io.Writer, field *form.Field) error {
	attrs := html.Attrs{"class": "form-group"}
	if field.Err() != nil {
		attrs["class"] += " has-error"
	}
	div := html.Div()
	div.Attrs = attrs
	div.Open = true
	_, err := div.WriteTo(w)
	if err == nil && field.Type == form.CHECKBOX {
		cb := html.Div()
		cb.Attrs = html.Attrs{"class": "checkbox"}
		cb.Open = true
		_, err = cb.WriteTo(w)
	}
	return err
}

func (r *FormRenderer) BeginLabel(w io.Writer, field *form.Field, label string, pos int) error {
	var err error
	if field.Type == form.CHECKBOX || field.Type == form.RADIO {
		if c := r.inlineLabelClass(); c != "" {
			div := html.Div()
			div.Attrs = html.Attrs{"class": c}
			div.Open = true
			_, err = div.WriteTo(w)
		}
	}
	if err == nil && field.Type == form.RADIO && pos >= 0 {
		div := html.Div()
		div.Attrs = html.Attrs{"class": "radio"}
		div.Open = true
		_, err = div.WriteTo(w)
	}
	return err
}

func (r *FormRenderer) LabelAttributes(field *form.Field, pos int) (html.Attrs, error) {
	if field.Type != form.CHECKBOX && field.Type != form.RADIO {
		if c := r.labelClass(); c != "" {
			return html.Attrs{"class": c}, nil
		}
	}
	return nil, nil
}

func (r *FormRenderer) EndLabel(w io.Writer, field *form.Field, pos int) error {
	var err error
	if field.Type == form.RADIO && pos >= 0 {
		_, err = io.WriteString(w, "</div>")
	}
	if err == nil {
		if field.Type != form.CHECKBOX && field.Type != form.RADIO {
			if c := r.inputDivClass(); c != "" {
				div := html.Div()
				div.Attrs = html.Attrs{"class": c}
				div.Open = true
				_, err = div.WriteTo(w)
			}
		}
	}
	return err
}

func (r *FormRenderer) BeginInput(w io.Writer, field *form.Field, placeholder string, pos int) error {
	var err error
	if field.HasAddOns() && pos == -1 {
		div := html.Div()
		div.Attrs = html.Attrs{"class": "input-group"}
		div.Open = true
		_, err = div.WriteTo(w)
	}
	return err
}

func (r *FormRenderer) FieldAttributes(field *form.Field, pos int) (html.Attrs, error) {
	if field.Type == form.CHECKBOX || (field.Type == form.SELECT && pos != -1) || field.Type == form.RADIO || field.Type == form.FILE {
		return nil, nil
	}
	return html.Attrs{
		"class": "form-control",
	}, nil
}

func (r *FormRenderer) EndInput(w io.Writer, field *form.Field, pos int) error {
	var err error
	if field.HasAddOns() && pos == -1 {
		_, err = io.WriteString(w, "</div>")
	}
	return err
}

func (r *FormRenderer) WriteAddOn(w io.Writer, field *form.Field, addon *form.AddOn) error {
	node := addon.Node
	if node.Type == html.TypeText {
		node = &html.Node{
			Tag:      "span",
			Attrs:    html.Attrs{"class": "input-group-addon"},
			Children: addon.Node,
		}
	}
	_, err := node.WriteTo(w)
	return err
}

func (r *FormRenderer) WriteError(w io.Writer, field *form.Field, err error) error {
	span := html.Span(html.Text(err.Error()))
	span.Attrs = html.Attrs{"class": "help-block"}
	_, werr := span.WriteTo(w)
	return werr
}

func (r *FormRenderer) WriteHelp(w io.Writer, field *form.Field, help string) error {
	span := html.Span(html.Text(help))
	span.Attrs = html.Attrs{"class": "help-block"}
	_, err := span.WriteTo(w)
	return err
}

func (r *FormRenderer) EndField(w io.Writer, field *form.Field) error {
	_, err := io.WriteString(w, "</div>")
	if err == nil && field.Type == form.CHECKBOX {
		_, err = io.WriteString(w, "</div>")
	}
	if err == nil && r.inputDivClass() != "" {
		_, err = io.WriteString(w, "</div>") // Close div with the columns
	}
	return err
}

func newRenderer() form.Renderer {
	return &FormRenderer{}
}

func init() {
	form.SetDefaultRenderer(newRenderer)
}
