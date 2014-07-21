package assets

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"

	"gnd.la/crypto/hashutil"
	"gnd.la/log"
)

var (
	compilers = map[Type]map[string]Compiler{}
)

type Compiler interface {
	Compile(w io.Writer, r io.Reader, opts Options) error
	Type() Type
	Ext() string
}

func RegisterCompiler(c Compiler) {
	typ := c.Type()
	ext := c.Ext()
	typeCompilers := compilers[typ]
	if typeCompilers == nil {
		typeCompilers = make(map[string]Compiler)
		compilers[typ] = typeCompilers
	}
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}
	typeCompilers[strings.ToLower(ext)] = c
}

func Compile(m *Manager, name string, typ Type, opts Options) (string, error) {
	ext := path.Ext(name)
	compiler := compilers[typ][strings.ToLower(ext)]
	if compiler == nil {
		return name, nil
	}
	f, err := m.Load(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	seeker, err := Seeker(f)
	fnv := hashutil.Fnv32a(seeker)
	out := fmt.Sprintf("%s.gen.%s.%s", name, fnv, typ.Ext())
	if o, _ := m.Load(out); o != nil {
		o.Close()
		log.Debugf("%s already compiled to %s", name, out)
		return out, nil
	}
	seeker.Seek(0, 0)
	var buf bytes.Buffer
	log.Debugf("compiling %s to %s", name, out)
	if err := compiler.Compile(&buf, seeker, opts); err != nil {
		return "", err
	}
	w, err := m.Create(out, true)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(w, bytes.NewReader(buf.Bytes())); err != nil {
		w.Close()
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return out, nil
}
