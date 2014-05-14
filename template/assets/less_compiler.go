package assets

import (
	"io"
	"os/exec"
)

var (
	lesscPath, _ = exec.LookPath("lessc")
)

type lessCompiler struct {
}

func (c *lessCompiler) Compile(w io.Writer, r io.Reader, opts Options) error {
	if lesscPath != "" {
		return command(lesscPath, []string{"--no-color", "-"}, w, r, opts)
	}
	_, _, err := reducer("less", w, r)
	return err
}

func (c *lessCompiler) Type() Type {
	return TypeCSS
}

func (c *lessCompiler) Ext() string {
	return "less"
}

func init() {
	RegisterCompiler(&lessCompiler{})
}
