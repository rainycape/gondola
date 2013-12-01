package assets

import (
	"bytes"
	"fmt"
	"gnd.la/log"
	"gnd.la/util/hashutil"
	"io"
	"path"
	"strings"
)

var (
	compilers = map[CodeType]map[string]Compiler{}
)

type Compiler interface {
	Compile(w io.Writer, r io.Reader, m Manager, opts Options) error
	CodeType() CodeType
	Ext() string
}

func RegisterCompiler(c Compiler) {
	codeType := c.CodeType()
	typeCompilers := compilers[codeType]
	if typeCompilers == nil {
		typeCompilers = make(map[string]Compiler)
		compilers[codeType] = typeCompilers
	}
	typeCompilers["."+strings.ToLower(c.Ext())] = c
}

func Compile(m Manager, name string, codeType CodeType, opts Options) (string, error) {
	ext := path.Ext(name)
	compiler := compilers[codeType][strings.ToLower(ext)]
	if compiler == nil {
		return name, nil
	}
	f, _, err := m.Load(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fnv := hashutil.Fnv32a(f)
	nonExt := name[:len(name)-len(ext)]
	out := fmt.Sprintf("%s.gen.%s.%s", nonExt, fnv, codeType.Ext())
	if _, _, err := m.Load(out); err == nil {
		log.Debugf("%s already compiled to %s", name, out)
		return out, nil
	}
	f.Seek(0, 0)
	var buf bytes.Buffer
	log.Debugf("compiling %s to %s", name, out)
	if err := compiler.Compile(&buf, f, m, opts); err != nil {
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
