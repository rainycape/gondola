package assets

import (
	"gnd.la/log"
	"io"
)

type cssBundler struct {
}

func (c *cssBundler) Bundle(w io.Writer, r io.Reader, opts Options) error {
	p, n, err := reducer("css", w, r)
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
