package assets

import (
	"gnd.la/log"
	"io"
)

// cssCompiler uses http://gondola-reducer.appspot.com/css to compile CSS code
type cssCompiler struct {
}

func (c *cssCompiler) Compile(r io.Reader, w io.Writer, m Manager, opts Options) error {
	p, n, err := reducer("css", w, r)
	if err != nil {
		return err
	}
	log.Debugf("Reduced CSS size from %d to %d bytes", p, n)
	return err
}

func (c *cssCompiler) CodeType() int {
	return CodeTypeCss
}

func (c *cssCompiler) Ext() string {
	return "css"
}

func (c *cssCompiler) Asset(name string, m Manager, opts Options) (Asset, error) {
	assets, err := cssParser(m, []string{name}, opts)
	if err != nil {
		return nil, err
	}
	return assets[0], nil
}

func init() {
	registerCompiler(&cssCompiler{}, true)
}
