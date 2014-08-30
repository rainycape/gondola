// Package json generates methods for encoding/decoding types to/from JSON.
//
// When used correctly, these methods can easily give a ~200-300% performance
// increase when serializing objects to JSON while also reducing memory usage
// by ~95-99%. For taking advantage of these gains, you must use
// gnd.la/app/serialize or Context.WriteJSON to encode to JSON, since
// json.Marshal won't use these methods correctly and might even have worse
// performance when these methods are implemented.
//
// This is a small benchmark comparing the performance of these JSON encoding
// methods. JSONDirect uses WriteJSON(), JSONSerialize uses
// gnd.la/app/serialize (which adds some overhead because it also sets the
// Content-Length and Content-Encoding headers and thus must encode into an
// intermediate buffer first), while JSON uses json.Marshal(). All three
// benchmarks write the result to ioutil.Discard.
//
//  BenchmarkJSONDirect	    1000000 1248 ns/op	117.73 MB/s 16 B/op	2 allocs/op
//  BenchmarkJSONSerialize  1000000 1587 ns/op	92.62 MB/s  16 B/op	2 allocs/op
//  BenchmarkJSON	    500000  4583 ns/op	32.07 MB/s  620 B/op	4 allocs/op
//
// Code generated by this package respects json related struct tags except
// omitempty and and encodes time.Time UTC as a Unix time (encoding/json uses
// time.Format).
//
// If you want to specify a different serialization when using encoding/json
// than when using this package, you can use the "genjson" field tag. Fields
// with a genjson tag will use it and ignore the "json" tag.
//
// The recommended way use to generate JSON methods for a given package is
// using the gondola command rather than using this package directly.
package json

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"code.google.com/p/go.tools/go/types"
	"gnd.la/internal/gen/genutil"
	"gnd.la/log"
	"gnd.la/util/generic"
	"gnd.la/util/structs"
)

const (
	defaultBufSize = 8 * 1024
)

// Field indicates a JSON field to be included in the output.
// Key indicates the key used in the JSON, while name indicate
// the field or method (in that case, it should receive no arguments)
// name.
type Field struct {
	Key       string
	Name      string
	OmitEmpty bool
}

// Options specify the options used when generating JSON related
// methods.
type Options struct {
	// Wheter to generate a MarshalJSON method. This is false by default
	// because in most cases will result in lower performance when using
	// json.Marshal, since the encoder from encoding/json will revalidate
	// the returned JSON, resulting in a performance loss. Turn this on
	// only if you're using the Methods feature (otherwise you'll get
	// different results when serializing with json.Marshal).
	MarshalJSON bool
	// The size of the allocated buffers for serializing to JSON. If zero,
	// the default size of 8192 is used (8K).
	BufferSize int
	// The maximum buffer size. Buffers which grow past this size won't
	// be reused. If zero, it takes the same value os BufferSize.
	MaxBufferSize int
	// The number of buffers to be kept for reusing. If zero, it defaults
	// to GOMAXPROCS. Set it to a negative number to disable buffering.
	BufferCount int
	// If not zero, this takes precedence over BufferCount. The number of
	// maximum buffers will be GOMAXPROCS * BuffersPerProc.
	BuffersPerProc int
	// TypeFields contains the per-type fields. The key in the map is the
	// type name in the package (e.g. MyStruct not mypackage.MyStruct).
	// Field tags are ignored for types that explicitely specify their
	// fields. Additionally, specific fields for contains might be
	// specified using the container.type syntax (e.g. MyOtherStruct.MyStruct
	// or MySliceType.MyStruct will set the fields for MyStruct only when
	// it's contained in MyOtherStruct or MySliceType respectively).
	// Any type in this map is included regardless of Include and Exclude.
	TypeFields map[string][]*Field
	// If not nil, only types matching this regexp will be included.
	Include *regexp.Regexp
	// If not nil, types matching this regexp will be excluded.
	Exclude *regexp.Regexp
}

// Gen generates a WriteJSON method and, optionally, MarshalJSON for every
// selected type in the given package. The package might be either an
// absolute path or an import path. See Options to learn how to select
// types and specify type options
func Gen(pkgName string, opts *Options) error {
	pkg, err := genutil.NewPackage(pkgName)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", pkg.Name()))
	buf.WriteString(genutil.AutogenString())
	buf.WriteString("\nimport (\n")
	imports := []string{"bytes", "io", "strconv", "unicode/utf8"}
	if opts == nil || opts.BufferCount == 0 || opts.BuffersPerProc != 0 {
		imports = append(imports, "runtime")
	}
	for _, v := range imports {
		buf.WriteString(fmt.Sprintf("%q\n", v))
	}
	buf.WriteString(")\n")
	buf.WriteString("var _ = strconv.FormatBool\n")
	buf.WriteString("var _ = io.ReadFull\n")
	var include *regexp.Regexp
	var exclude *regexp.Regexp
	var names []string
	if opts != nil {
		include = opts.Include
		exclude = opts.Exclude

		for k := range opts.TypeFields {
			names = append(names, strings.Split(k, ".")...)
		}
	}
	typs, err := pkg.SelectedTypes(include, exclude, names)
	if err != nil {
		return err
	}
	var methods bytes.Buffer
	for _, v := range typs {
		methods.Reset()
		log.Debugf("generating JSON methods for %s", v.Obj().Name())
		if err := jsonMarshal(v, opts, &methods); err != nil {
			return fmt.Errorf("error in type %s: %s", v.Obj().Name(), err)
		}
		buf.Write(methods.Bytes())
	}
	buf.WriteString(encode_go)
	bufSize := defaultBufSize
	maxBufSize := bufSize
	bufferCount := 0
	buffersPerProc := 0
	if opts != nil {
		if opts.BufferSize > 0 {
			bufSize = opts.BufferSize
			maxBufSize = bufSize
		}
		if opts.MaxBufferSize >= maxBufSize {
			maxBufSize = opts.MaxBufferSize
		}
		bufferCount = opts.BufferCount
		buffersPerProc = opts.BuffersPerProc
	}
	buf.WriteString(fmt.Sprintf("const jsonBufSize = %d\n", bufSize))
	buf.WriteString(fmt.Sprintf("const jsonMaxBufSize = %d\n", maxBufSize))
	if buffersPerProc > 0 {
		buf.WriteString(fmt.Sprintf("var jsonBufferCount = runtime.GOMAXPROCS(0) * %d\n", buffersPerProc))
	} else if bufferCount > 0 {
		buf.WriteString(fmt.Sprintf("const jsonBufferCount = %d\n", bufferCount))
	} else {
		buf.WriteString("var jsonBufferCount = runtime.GOMAXPROCS(0)\n")
	}
	buf.WriteString(buffer_go)
	out := filepath.Join(pkg.Dir(), "gen_json.go")
	log.Debugf("Writing autogenerated JSON methods to %s", out)
	return genutil.WriteAutogen(out, buf.Bytes())
}

func jsonMarshal(typ *types.Named, opts *Options, buf *bytes.Buffer) error {
	tname := typ.Obj().Name()
	if _, ok := typ.Underlying().(*types.Struct); ok {
		tname = "*" + tname
	}
	if opts != nil && opts.MarshalJSON {
		buf.WriteString(fmt.Sprintf("func(o %s) MarshalJSON() ([]byte, error) {\n", tname))
		buf.WriteString("var buf bytes.Buffer\n")
		buf.WriteString("_, err := o.WriteJSON(&buf)\n")
		buf.WriteString("return buf.Bytes(), err\n")
		buf.WriteString("}\n\n")
	}
	buf.WriteString(fmt.Sprintf("func(o %s) WriteJSON(w io.Writer) (int, error) {\n", tname))
	buf.WriteString("buf := jsonGetBuffer()\n")
	if err := jsonValue(typ, nil, "o", opts, buf); err != nil {
		return err
	}
	buf.WriteString("n, err := w.Write(buf.Bytes())\n")
	buf.WriteString("jsonPutBuffer(buf)\n")
	buf.WriteString("return n, err\n")
	buf.WriteString("}\n\n")
	return nil
}

func fieldTag(tag string) *structs.Tag {
	if gtag := structs.NewStringTagNamed(tag, "genjson"); gtag != nil && !gtag.IsEmpty() {
		return gtag
	}
	return structs.NewStringTagNamed(tag, "json")
}

func fieldByName(st *types.Struct, name string) *types.Var {
	count := st.NumFields()
	for ii := 0; ii < count; ii++ {
		field := st.Field(ii)
		if field.Name() == name {
			return field
		}
	}
	return nil
}

func methodByName(tn *types.Named, name string) (*types.Var, error) {
	count := tn.NumMethods()
	for ii := 0; ii < count; ii++ {
		fn := tn.Method(ii)
		if fn.Name() == name {
			signature := fn.Type().(*types.Signature)
			if p := signature.Params(); p != nil || p.Len() > 1 {
				fmt.Println("SIGN", signature, p, p.Len())
				return nil, fmt.Errorf("method %s on type %s requires arguments", name, tn.Obj().Name())
			}
			res := signature.Results()
			if res == nil || res.Len() != 1 {
				return nil, fmt.Errorf("method %s on type %s must return exactly one value", name, tn.Obj().Name())
			}
			return res.At(0), nil
		}
	}
	return nil, nil
}

func namesSelectors(names []string) []string {
	if len(names) <= 1 {
		return names
	}
	selectors := []string{strings.Join(names, ".")}
	for ii := 0; ii < len(names)-1; ii++ {
		var cur []string
		cur = append(cur, names[:ii]...)
		cur = append(cur, names[ii+1:]...)
		selectors = append(selectors, namesSelectors(cur)...)
	}
	generic.SortFunc(selectors, func(s1, s2 string) bool {
		c1 := strings.Count(s1, ".")
		c2 := strings.Count(s2, ".")
		if c1 != c2 {
			return c1 > c2
		}
		suf1 := strings.HasSuffix(selectors[0], s1)
		suf2 := strings.HasSuffix(selectors[0], s2)
		if suf1 != suf2 {
			return suf1
		}
		return len(s1) > len(s2)
	})
	return selectors
}

func typeSelectors(typs []types.Type) []string {
	var names []string
	for _, v := range typs {
		if n, ok := v.(*types.Named); ok {
			names = append(names, n.Obj().Name())
		}
	}
	return namesSelectors(names)
}

func appendType(typs []types.Type, typ types.Type) []types.Type {
	var t []types.Type
	t = append(t, typs...)
	t = append(t, typ)
	return t
}

func expandStructFields(st *types.Struct, fields []*Field, opts *Options) ([]*Field, error) {
	var newFields []*Field
	keys := make(map[string]bool)
	// First check if there's any expansion
	for _, f := range fields {
		if f.Key == "+" {
			names := strings.Split(f.Name, ",")
			for _, n := range names {
				n = strings.TrimSpace(n)
				switch strings.ToLower(n) {
				case "fields":
					newFields = append(newFields, defaultStructFields(st, opts)...)
				default:
					return nil, fmt.Errorf("unknown expansion %q", n)
				}
			}
		}
	}
	if len(newFields) == 0 {
		// no expansions or expansion without effect
		return fields, nil
	}
	// Add remaining fields
	for _, f := range fields {
		if f.Key == "+" || keys[f.Key] {
			continue
		}
		newFields = append(newFields, f)
	}
	return newFields, nil
}

func defaultStructFields(st *types.Struct, opts *Options) []*Field {
	var fields []*Field
	count := st.NumFields()
	for ii := 0; ii < count; ii++ {
		field := st.Field(ii)
		key := field.Name()
		omitEmpty := false
		tag := st.Tag(ii)
		if ftag := fieldTag(tag); ftag != nil {
			if n := ftag.Name(); n != "" {
				key = n
			}
			omitEmpty = ftag.Has("omitempty")
		} else if !field.Exported() {
			continue
		}
		if key != "-" {
			fields = append(fields, &Field{Key: key, Name: field.Name(), OmitEmpty: omitEmpty})
		}
	}
	return fields
}

func jsonStruct(st *types.Struct, parents []types.Type, name string, opts *Options, buf *bytes.Buffer) error {
	buf.WriteString("buf.WriteByte('{')\n")
	var named *types.Named
	if len(parents) > 0 {
		p := parents[len(parents)-1]
		if n, ok := p.(*types.Named); ok {
			named = n
		}
	}
	var fields []*Field
	if opts != nil {
		for _, v := range typeSelectors(parents) {
			if fields = opts.TypeFields[v]; fields != nil {
				break
			}
		}
	}
	if fields == nil {
		fields = defaultStructFields(st, opts)
	} else {
		var err error
		fields, err = expandStructFields(st, fields, opts)
		if err != nil {
			return fmt.Errorf("error expanding fields for type %s: %s", name, err)
		}
	}
	typs := appendType(parents, st)
	for ii, v := range fields {
		field := fieldByName(st, v.Name)
		var suffix string
		if field == nil && named != nil {
			var err error
			field, err = methodByName(named, v.Name)
			if err != nil {
				return err
			}
			suffix = "()"
		}
		if field == nil {
			var t types.Type = st
			if named != nil {
				t = named
			}
			return fmt.Errorf("type %s does not have a field nor method called %q", t, v.Name)
		}
		if ii > 0 {
			buf.WriteString("buf.WriteByte(',')\n")
		}
		if err := jsonField(field, typs, v.Key, name+"."+v.Name+suffix, v.OmitEmpty, opts, buf); err != nil {
			return err
		}
	}
	buf.WriteString("buf.WriteByte('}')\n")
	return nil
}

func jsonSlice(sl *types.Slice, parents []types.Type, name string, opts *Options, buf *bytes.Buffer) error {
	buf.WriteString("buf.WriteByte('[')\n")
	buf.WriteString(fmt.Sprintf("for ii, v := range %s {\n", name))
	buf.WriteString("if ii > 0 {\n")
	buf.WriteString("buf.WriteByte(',')\n")
	buf.WriteString("}\n")
	if err := jsonValue(sl.Elem(), parents, "v", opts, buf); err != nil {
		return err
	}
	buf.WriteString("}\n")
	buf.WriteString("buf.WriteByte(']')\n")
	return nil
}

func jsonField(field *types.Var, parents []types.Type, key string, name string, omitEmpty bool, opts *Options, buf *bytes.Buffer) error {
	// TODO: omitEmpty
	buf.WriteString(fmt.Sprintf("buf.WriteString(%q)\n", fmt.Sprintf("%q", key)))
	buf.WriteString("buf.WriteByte(':')\n")
	if err := jsonValue(field.Type(), parents, name, opts, buf); err != nil {
		return err
	}
	return nil
}

func jsonValue(vtype types.Type, parents []types.Type, name string, opts *Options, buf *bytes.Buffer) error {
	switch typ := vtype.(type) {
	case *types.Basic:
		k := typ.Kind()
		var isPointer bool
		if len(parents) > 0 {
			_, isPointer = parents[len(parents)-1].(*types.Pointer)
		}
		if isPointer {
			name = "*" + name
		}
		switch k {
		case types.Bool:
			fmt.Fprintf(buf, "buf.WriteString(strconv.FormatBool(%s))\n", name)
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			fmt.Fprintf(buf, "buf.WriteString(strconv.FormatInt(int64(%s), 10))\n", name)
		case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			fmt.Fprintf(buf, "buf.WriteString(strconv.FormatUint(uint64(%s), 10))\n", name)
		case types.Float32, types.Float64:
			bitSize := 64
			if k == types.Float32 {
				bitSize = 32
			}
			fmt.Fprintf(buf, "buf.WriteString(strconv.FormatFloat(float64(%s), 'g', -1, %d))\n", name, bitSize)
		case types.String:
			fmt.Fprintf(buf, "jsonEncodeString(buf, string(%s))\n", name)
		default:
			return fmt.Errorf("can't encode basic kind %v", typ.Kind())
		}
	case *types.Named:
		if typ.Obj().Pkg().Name() == "time" && typ.Obj().Name() == "Time" {
			fmt.Fprintf(buf, "buf.WriteString(strconv.FormatInt(%s.UTC().Unix(), 10))\n", name)
		} else {
			if err := jsonValue(typ.Underlying(), appendType(parents, typ), name, opts, buf); err != nil {
				return err
			}
		}
	case *types.Slice:
		if err := jsonSlice(typ, parents, name, opts, buf); err != nil {
			return err
		}
	case *types.Struct:
		if err := jsonStruct(typ, parents, name, opts, buf); err != nil {
			return err
		}
	case *types.Pointer:
		buf.WriteString(fmt.Sprintf("if %s == nil {\n", name))
		buf.WriteString("buf.WriteString(\"null\")\n")
		buf.WriteString("} else {\n")
		if err := jsonValue(typ.Elem(), appendType(parents, typ), name, opts, buf); err != nil {
			return err
		}
		buf.WriteString("}\n")
	default:
		return fmt.Errorf("can't encode type %T %v (%T)", typ, typ, typ.Underlying())
	}
	return nil
}
