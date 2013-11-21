package assets

import (
	"io"
)

var (
	bundlers = map[CodeType]Bundler{}
)

func RegisterBundler(b Bundler) {
	codeType := b.CodeType()
	bundlers[codeType] = b
}

type Bundler interface {
	Bundle(w io.Writer, r io.Reader, m Manager, opts Options) error
	CodeType() CodeType
	Asset(name string, m Manager, opts Options) (Asset, error)
}
