package assets

import (
	"io"
)

var (
	bundlers = map[Type]Bundler{}
)

func RegisterBundler(b Bundler) {
	bundlers[b.Type()] = b
}

type Bundler interface {
	Bundle(w io.Writer, r io.Reader, opts Options) error
	Type() Type
}
