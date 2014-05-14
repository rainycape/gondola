package assets

import (
	"io"
	"os/exec"
)

var (
	coffeePath, _ = exec.LookPath("coffee")
)

type coffeeCompiler struct {
}

func (c *coffeeCompiler) Compile(w io.Writer, r io.Reader, opts Options) error {
	if coffeePath != "" {
		return command(coffeePath, []string{"-sc", "-"}, w, r, opts)
	}
	_, _, err := assetsService("coffee", w, r)
	return err
}

func (c *coffeeCompiler) Type() Type {
	return TypeJavascript
}

func (c *coffeeCompiler) Ext() string {
	return "coffee"
}

func init() {
	RegisterCompiler(&coffeeCompiler{})
}
