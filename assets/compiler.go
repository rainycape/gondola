package assets

import (
	"fmt"
	"io"
)

var (
	compilers = map[int]Compiler{}
)

func RegisterCompiler(c Compiler) {
	registerCompiler(c, false)
}

func registerCompiler(c Compiler, internal bool) {
	// Let users override the default compilers
	codeType := c.CodeType()
	if !internal && codeType <= 1000 && compilers[codeType] == nil {
		panic(fmt.Errorf("invalid custom code type %d (must be > 1000)", codeType))
	}
	compilers[codeType] = c
}

type Compiler interface {
	Compile(r io.Reader, w io.Writer, m Manager, opts Options) error
	CodeType() int
	Ext() string
	Asset(name string, m Manager, opts Options) (Asset, error)
}
