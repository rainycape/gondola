package assets

import (
	"io"
	"os/exec"

	"gnd.la/log"
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
	p, n, err := assetsService("css", w, r)
	if err != nil {
		return err
	}
	log.Debugf("Reduced CSS size from %d to %d bytes", p, n)
	return err
}

func (c *cssBundler) Type() Type {
	return TypeCSS
}

func init() {
	RegisterBundler(&cssBundler{})
}
