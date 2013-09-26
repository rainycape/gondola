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
	if err == nil && field.Type == form.CHECKBOX {
		cb := html.Div()
		cb.Attrs = html.Attrs{"class": "checkbox"}
		cb.Open = true
		_, err = cb.WriteTo(w)
	}
	return err
}

func (r *Renderer) BeginLabel(w io.Writer, field *form.Field, pos int) error {
	var err error
	if field.Type == form.RADIO && pos >= 0 {
		div := html.Div()
		div.Attrs = html.Attrs{"class": "radio"}
		div.Open = true
		_, err = div.WriteTo(w)
	}
	return err
}

func (r *Renderer) LabelAttributes(field *form.Field, pos int) (html.Attrs, error) {
	return nil, nil
}

func (r *Renderer) EndLabel(w io.Writer, field *form.Field, pos int) error {
	var err error
	if field.Type == form.RADIO && pos >= 0 {
		_, err = io.WriteString(w, "</div>")
	}
	return err
}

func (r *Renderer) BeginInput(w io.Writer, field *form.Field, pos int) error {
	var err error
	if field.HasAddOns() && pos == -1 {
		div := html.Div()
		div.Attrs = html.Attrs{"class": "input-group"}
		div.Open = true
		_, err = div.WriteTo(w)
	}
	return err
}

func (r *Renderer) FieldAttributes(field *form.Field, pos int) (html.Attrs, error) {
	if field.Type == form.CHECKBOX || (field.Type == form.SELECT && pos != -1) || field.Type == form.RADIO {
		return nil, nil
	}
	return html.Attrs{
		"class": "form-control",
	}, nil
}

func (r *Renderer) EndInput(w io.Writer, field *form.Field, pos int) error {
	var err error
	if field.HasAddOns() && pos == -1 {
		_, err = io.WriteString(w, "</div>")
	}
	return err
}

func (r *Renderer) WriteAddOn(w io.Writer, field *form.Field, addon *form.AddOn) error {
	node := addon.Node
	if node.Type == html.TEXT_NODE {
		node = &html.Node{
			Tag:      "span",
			Attrs:    html.Attrs{"class": "input-group-addon"},
			Children: addon.Node,
		}
	}
	_, err := node.WriteTo(w)
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
	if err == nil && field.Type == form.CHECKBOX {
		_, err = io.WriteString(w, "</div>")
	}
	return err
}
