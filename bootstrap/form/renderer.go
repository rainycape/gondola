package form

import (
	"gnd.la/form"
	"gnd.la/html"
	"io"
)

type Renderer struct {
}

func (r *Renderer) BeginField(w io.Writer, field *form.Field) error {
	attrs := html.Attrs{"class": "form-group"}
	if field.Err() != nil {
		attrs["class"] += " has-error"
	}
	div := html.Div()
	div.Attrs = attrs
	div.Open = true
	_, err := div.WriteTo(w)
	if err == nil && field.Type == form.BOOL {
		cb := html.Div()
		cb.Attrs = html.Attrs{"class": "checkbox"}
		cb.Open = true
		_, err = cb.WriteTo(w)
	}
	return err
}

func (r *Renderer) LabelAttributes(field *form.Field) (html.Attrs, error) {
	return nil, nil
}

func (r *Renderer) BeginInput(w io.Writer, field *form.Field) error {
	var err error
	if field.HasAddOns() {
		div := html.Div()
		div.Attrs = html.Attrs{"class": "input-group"}
		div.Open = true
		_, err = div.WriteTo(w)
	}
	return err
}

func (r *Renderer) FieldAttributes(field *form.Field) (html.Attrs, error) {
	if field.Type == form.BOOL {
		return nil, nil
	}
	return html.Attrs{
		"class": "form-control",
	}, nil
}

func (r *Renderer) EndInput(w io.Writer, field *form.Field) error {
	var err error
	if field.HasAddOns() {
		_, err = io.WriteString(w, "</div>")
	}
	return err
}

func (r *Renderer) WriteAddOn(w io.Writer, field *form.Field, addon *form.AddOn) error {
	span := &html.Node{
		Tag:      "span",
		Attrs:    html.Attrs{"class": "input-group-addon"},
		Children: addon.Node,
	}
	_, err := span.WriteTo(w)
	return err
}

func (r *Renderer) WriteError(w io.Writer, field *form.Field, err error) error {
	span := html.Span(html.Text(err.Error()))
	span.Attrs = html.Attrs{"class": "help-block"}
	_, werr := span.WriteTo(w)
	return werr
}

func (r *Renderer) WriteHelp(w io.Writer, field *form.Field) error {
	span := html.Span(html.Text(field.Help))
	span.Attrs = html.Attrs{"class": "help-block"}
	_, err := span.WriteTo(w)
	return err
}

func (r *Renderer) EndField(w io.Writer, field *form.Field) error {
	_, err := io.WriteString(w, "</div>")
	if err == nil && field.Type == form.BOOL {
		_, err = io.WriteString(w, "</div>")
	}
	return err
}
