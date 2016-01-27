// Package sass implements a sass compiler for assets.
package sass

import (
	"bytes"
	"io"
	"path/filepath"

	"gnd.la/template/assets"

	"github.com/wellington/go-libsass"
)

// implemented by os.File
type named interface {
	Name() string
}

type sassCompiler struct {
}

func (c *sassCompiler) Compile(w io.Writer, r io.Reader, opts assets.Options) error {
	var includePaths []string
	if n, ok := r.(named); ok {
		name := n.Name()
		dir := filepath.Dir(name)
		includePaths = append(includePaths, dir)
	}
	var buf bytes.Buffer
	comp, err := libsass.New(&buf, r, libsass.IncludePaths(includePaths))
	if err != nil {
		return err
	}
	if err := comp.Run(); err != nil {
		return err
	}
	_, err = io.Copy(w, &buf)
	return err
}

func (c *sassCompiler) Type() assets.Type {
	return assets.TypeCSS
}

func (c *sassCompiler) Ext() string {
	return "scss"
}

func init() {
	assets.RegisterCompiler(&sassCompiler{})
}
