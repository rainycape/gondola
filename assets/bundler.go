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
	Bundle(r io.Reader, w io.Writer, m Manager, opts Options) error
	CodeType() CodeType
	Ext() string
	Asset(name string, m Manager, opts Options) (Asset, error)
}
