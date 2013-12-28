package assets

import (
	"gnd.la/log"
	"io"
)

type cssBundler struct {
}

func (c *cssBundler) Bundle(w io.Writer, r io.Reader, m *Manager, opts Options) error {
	p, n, err := reducer("css", w, r)
	if err != nil {
		return err
	}
	log.Debugf("Reduced CSS size from %d to %d bytes", p, n)
	return err
}

func (c *cssBundler) CodeType() CodeType {
	return CodeTypeCss
}

func (c *cssBundler) Asset(name string, m *Manager, opts Options) (*Asset, error) {
	return CSS(name, m.URL(name)), nil
}

func init() {
	RegisterBundler(&cssBundler{})
}
