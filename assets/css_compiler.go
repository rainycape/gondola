package assets

import (
	"fmt"
	"gnd.la/log"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// cssCompiler uses http://gondola-reducer.appspot.com/css to compile CSS code
type cssCompiler struct {
}

func (c *cssCompiler) Compile(r io.Reader, w io.Writer, m Manager, opts Options) error {
	code, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	form := url.Values{
		"file": []string{string(code)},
	}
	resp, err := http.PostForm("http://gondola-reducer.appspot.com/css", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("invalid CSS code: %s", string(msg))
	}
	n, err := io.Copy(w, resp.Body)
	log.Debugf("Reduced CSS size from %d to %d bytes", len(code), n)
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
