package assets

import (
	"gnd.la/log"
	"io"
)

type cssBundler struct {
}

func (c *cssBundler) Bundle(r io.Reader, w io.Writer, m Manager, opts Options) error {
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

func (c *cssBundler) Ext() string {
	return "css"
}

func (c *cssBundler) Asset(name string, m Manager, opts Options) (Asset, error) {
	assets, err := cssParser(m, []string{name}, opts)
	if err != nil {
		return nil, err
	}
	return assets[0], nil
}

func init() {
	RegisterBundler(&cssBundler{})
}
