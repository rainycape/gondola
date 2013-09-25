package form

import (
	"bytes"
	"fmt"
	"gnd.la/html"
	"gnd.la/i18n"
	"gnd.la/mux"
	"gnd.la/password"
	"gnd.la/types"
	"gnd.la/util"
	"html/template"
	"reflect"
	"strconv"
)

var (
	formTags = []string{"form", "gondola"}
)

type attrMap map[string]html.Attrs

type Form struct {
	ctx            *mux.Context
	renderer       Renderer
	values         []reflect.Value
	structs        []*types.Struct
	fields         []*Field
	attrs          attrMap
	options        *Options
	invalid        bool
	validated      bool
	NamelessErrors bool
}

func (f *Form) validate() {
	for _, v := range f.fields {
		input := f.ctx.FormValue(v.Name)
		label := v.Label
		if f.NamelessErrors {
			label = ""
		}
		if err := types.InputNamed(label, input, v.SettableValue(), v.Tag(), true); err != nil {
			v.err = i18n.TranslatedError(err, f.ctx)
			f.invalid = true
			continue
		}
		if err := types.Validate(v.sval.Addr().Interface(), v.GoName, f.ctx); err != nil {
			v.err = i18n.TranslatedError(err, f.ctx)
			f.invalid = true
			continue
		}
	}
}

func (f *Form) fieldByName(id, name string) (*Field, error) {
	var s *types.Struct
	idx := -1
	var fieldValue reflect.Value
	var sval reflect.Value
	for ii, v := range f.structs {
		pos, ok := v.QNameMap[name]
		if ok {
			if s != nil {
				return nil, fmt.Errorf("duplicate field %q (found in %v and %v)", name, s.Type, v.Type)
			}
			s = v
			idx = pos
			sval = f.values[ii]
			fieldValue = sval.FieldByIndex(s.Indexes[pos])
			// Check the validation function, so if the function is not valid
			// the error is generated at form instantiation.
			if _, err := types.ValidationFunction(sval, name); err != nil {
				return nil, err
			}
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("can't map form field %q", name)
	}
	mangled := util.CamelCaseToLower(name, "_")
	tag := s.Tags[idx]
	label := tag.Value("label")
	if label == "" {
		label = util.CamelCaseToWords(name, " ")
	}
	var typ Type
	if tag.Has("hidden") {
		typ = HIDDEN
	} else {
		switch s.Types[idx].Kind() {
		case reflect.String:
			if s.Types[idx] == reflect.TypeOf(password.Password("")) {
				typ = PASSWORD
			} else if tag.Has("hidden") {
				typ = HIDDEN
			} else {
				if ml, ok := tag.MaxLength(); ok && ml > 0 {
					typ = TEXT
				} else if tag.Has("singleline") {
					typ = TEXT
				} else if tag.Has("password") {
					typ = PASSWORD
				} else {
					typ = TEXTAREA
				}
			}
		case reflect.Bool:
			typ = BOOL
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			typ = INTEGER
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			typ = UINTEGER
		case reflect.Float32, reflect.Float64:
			typ = FLOAT
		}
	}
	field := &Field{
		Id:          id + mangled,
		Type:        typ,
		Label:       label,
		GoName:      name,
		Name:        mangled,
		Placeholder: tag.Value("placeholder"),
		Help:        tag.Value("help"),
		value:       fieldValue,
		s:           s,
		sval:        sval,
		pos:         idx,
	}
	return field, nil
}

func (f *Form) lookupField(name string) (*Field, error) {
	for _, v := range f.fields {
		if v.GoName == name {
			return v, nil
		}
	}
	return nil, fmt.Errorf("form has no field named %q", name)
}

func (f *Form) makeFields(names []string) error {
	// Use the form pointer as part of the id, to ensure
	// uniqueness
	id := fmt.Sprintf("%p_", f)
	fields := make([]*Field, len(names))
	var err error
	for ii, v := range names {
		fields[ii], err = f.fieldByName(id, v)
		if err != nil {
			return err
		}
	}
	f.fields = fields
	return nil
}

func (f *Form) appendVal(val interface{}) error {
	v, err := types.SettableValue(val)
	if err != nil {
		return err
	}
	s, err := types.NewStruct(val, formTags)
	if err != nil {
		return err
	}
	f.values = append(f.values, v)
	f.structs = append(f.structs, s)
	return nil
}

func (f *Form) HasErrors() bool {
	return f.Submitted() && !f.IsValid()
}

func (f *Form) Submitted() bool {
	return f.ctx.R.Method == "POST" || f.ctx.FormValue("submitted") != ""
}

func (f *Form) IsValid() bool {
	if !f.validated {
		f.validate()
		f.validated = true
	}
	return !f.invalid
}

func (f *Form) Fields() []*Field {
	return f.fields
}

func (f *Form) FieldNames() []string {
	names := make([]string, len(f.fields))
	for ii, v := range f.fields {
		names[ii] = v.GoName
	}
	return names
}

func (f *Form) Renderer() Renderer {
	return f.renderer
}

func (f *Form) writeTag(buf *bytes.Buffer, tag string, attrs html.Attrs, closed bool) {
	buf.WriteByte('<')
	if closed {
		buf.WriteByte('/')
		buf.WriteString(tag)
	} else {
		buf.WriteString(tag)
		if attrs != nil {
			attrs.WriteTo(buf)
		}
	}
	buf.WriteByte('>')
}

func (f *Form) openTag(buf *bytes.Buffer, tag string, attrs html.Attrs) {
	f.writeTag(buf, tag, attrs, false)
}

func (f *Form) closeTag(buf *bytes.Buffer, tag string) {
	f.writeTag(buf, tag, nil, true)
}

func (f *Form) writeField(buf *bytes.Buffer, field *Field) error {
	var closed bool
	if field.Type != HIDDEN {
		closed = field.Type != BOOL
		if err := f.writeLabel(buf, field, closed); err != nil {
			return err
		}
		if r := f.renderer; r != nil {
			if err := r.BeginInput(buf, field); err != nil {
				return err
			}
			for _, a := range field.addons {
				if a.Position == AddOnPositionBefore {
					err := r.WriteAddOn(buf, field, a)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	var err error
	switch field.Type {
	case TEXT, INTEGER, UINTEGER, FLOAT:
		err = f.writeInput(buf, "text", field)
	case PASSWORD:
		err = f.writeInput(buf, "password", field)
	case HIDDEN:
		err = f.writeInput(buf, "hidden", field)
	case TEXTAREA:
		attrs := html.Attrs{
			"id":   field.Id,
			"name": field.Name,
		}
		if _, ok := field.Tag().IntValue("rows"); ok {
			attrs["rows"] = field.Tag().Value("rows")
		}
		if f.renderer != nil {
			fattrs, err := f.renderer.FieldAttributes(field)
			if err != nil {
				return err
			}
			for k, v := range fattrs {
				attrs[k] = v
			}
		}
		f.openTag(buf, "textarea", attrs)
		buf.WriteString(types.ToString(field.Value()))
		f.closeTag(buf, "textarea")
	case BOOL:
		err = f.writeInput(buf, "checkbox", field)
	case CHOICES:
	case SELECT:
	}
	if field.Type != HIDDEN && !closed {
		buf.WriteString(html.Escape(field.Label))
		f.closeTag(buf, "label")
	}
	return err
}

func (f *Form) writeLabel(buf *bytes.Buffer, field *Field, closed bool) error {
	attrs := html.Attrs{}
	if f.renderer != nil {
		lattrs, err := f.renderer.LabelAttributes(field)
		if err != nil {
			return err
		}
		for k, v := range lattrs {
			attrs[k] = v
		}
	}
	if field.Id != "" {
		attrs["for"] = field.Id
	}
	f.openTag(buf, "label", attrs)
	if closed {
		buf.WriteString(html.Escape(field.Label))
		f.closeTag(buf, "label")
	}
	return nil
}

func (f *Form) writeInput(buf *bytes.Buffer, itype string, field *Field) error {
	attrs := html.Attrs{
		"id":   field.Id,
		"type": itype,
		"name": field.Name,
	}
	if f.renderer != nil {
		fattrs, err := f.renderer.FieldAttributes(field)
		if err != nil {
			return err
		}
		for k, v := range fattrs {
			attrs[k] = v
		}
	}
	switch field.Type {
	case BOOL:
		if t, ok := types.IsTrue(field.value.Interface()); t && ok {
			attrs["checked"] = "checked"
		}
	case CHOICES:
	default:
		attrs["value"] = html.Escape(types.ToString(field.Value()))
		if field.Placeholder != "" {
			attrs["placeholder"] = field.Placeholder
		}
		if ml, ok := field.Tag().MaxLength(); ok {
			attrs["maxlength"] = strconv.Itoa(ml)
		}
	}
	f.openTag(buf, "input", attrs)
	return nil
}

func (f *Form) renderField(buf *bytes.Buffer, field *Field) (err error) {
	if provider, ok := field.sval.Addr().Interface().(AddOnProvider); ok {
		field.addons = provider.FieldAddOns(field)
	}
	r := f.renderer
	if r != nil {
		err = r.BeginField(buf, field)
		if err != nil {
			return
		}
	}
	err = f.writeField(buf, field)
	if err != nil {
		return
	}
	if r != nil {
		for _, a := range field.addons {
			if a.Position == AddOnPositionAfter {
				err = r.WriteAddOn(buf, field, a)
				if err != nil {
					return
				}
			}
		}
		err = r.EndInput(buf, field)
		if err != nil {
			return err
		}
		if ferr := field.Err(); ferr != nil {
			err = r.WriteError(buf, field, ferr)
			if err != nil {
				return
			}
		}
		if field.Help != "" {
			err = r.WriteHelp(buf, field)
			if err != nil {
				return
			}
		}
		err = r.EndField(buf, field)
		if err != nil {
			return
		}
	}
	return
}

func (f *Form) render(fields []*Field) (template.HTML, error) {
	var buf bytes.Buffer
	var err error
	for _, v := range fields {
		if err = f.renderField(&buf, v); err != nil {
			break
		}
	}
	return template.HTML(buf.String()), err
}

func (f *Form) Render() (template.HTML, error) {
	return f.render(f.fields)
}

func (f *Form) RenderOnly(names ...string) (template.HTML, error) {
	var fields []*Field
	for _, v := range names {
		field, err := f.lookupField(v)
		if err != nil {
			return template.HTML(""), err
		}
		fields = append(fields, field)
	}
	return f.render(fields)
}

func (f *Form) RenderExcept(names ...string) (template.HTML, error) {
	n := make(map[string]bool, len(names))
	for _, v := range names {
		n[v] = true
	}
	var fields []*Field
	for _, v := range f.fields {
		if !n[v.GoName] {
			fields = append(fields, v)
		}
	}
	return f.render(fields)
}

// New returns a new form using the given context, renderer and options. If render is
// nil, BasicRenderer will be used. The values argument must contains pointers to structs.
// Since any error generated during form creation will be a programming error, New panics
// rather than returning it. This way chaining is also possible. Consult the package
// documentation for the the tags parsed by the form library.
// Gondola also contains specific renderers for Bootstrap and Foundation, check the
// gnd.la/foundation and gnd.la/bootstrap packages for more information.
func New(ctx *mux.Context, r Renderer, opt *Options, values ...interface{}) *Form {
	form := &Form{
		ctx:      ctx,
		renderer: r,
		options:  opt,
	}
	for _, v := range values {
		err := form.appendVal(v)
		if err != nil {
			panic(err)
		}
	}
	var fieldNames []string
	if opt != nil && len(opt.Fields) > 0 {
		fieldNames = opt.Fields
	} else {
		for _, v := range form.structs {
			fieldNames = append(fieldNames, v.QNames...)
		}
	}
	err := form.makeFields(fieldNames)
	if err != nil {
		panic(err)
	}
	return form
}
