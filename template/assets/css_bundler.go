package assets

import (
	"io"
	"os/exec"
)

var (
	cleanCSSPath, _ = exec.LookPath("cleancss")
)

type cssBundler struct {
}

func (c *cssBundler) Bundle(w io.Writer, r io.Reader, opts Options) error {
	if cleanCSSPath != "" {
		return command(cleanCSSPath, []string{"--s0"}, w, r, opts)
	}
	_, _, err := assetsService("css", w, r)
	return err
}

func (c *cssBundler) Type() Type {
	return TypeCSS
}

func init() {
	RegisterBundler(&cssBundler{})
}
