package assets

import (
	"io"
)

type lessCompiler struct {
}

func (c *lessCompiler) Compile(w io.Writer, r io.Reader, m Manager, opts Options) error {
	_, _, err := reducer("less", w, r)
	return err
}

func (c *lessCompiler) CodeType() CodeType {
	return CodeTypeCss
}

func (c *lessCompiler) Ext() string {
	return "less"
}

func init() {
	RegisterCompiler(&lessCompiler{})
}
