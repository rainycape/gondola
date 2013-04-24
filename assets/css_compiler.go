package assets

import (
	"fmt"
	"gondola/log"
	"io/ioutil"
	"net/http"
	"net/url"
)

func cssCompiler(m Manager, name string, assets []CodeAsset, o Options) error {
	code, err := Code(assets)
	if err != nil {
		return err
	}
	form := url.Values{
		"file": []string{code},
	}
	resp, err := http.PostForm("http://reducisaurus.appspot.com/css", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Invalid status code from CSS compiler: %d", resp.StatusCode)
	}
	f, err := m.Create(name)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	defer f.Close()
	log.Debugf("Reduced CSS size from %d to %d", len(code), len(data))
	return nil
}

// CssCompiler uses http://reducisaurus.appspot.com/css to compile CSS code
func CssCompiler(m Manager, assets []CodeAsset, o Options) ([]Asset, error) {
	name := CodeAssetList(assets).CompiledName("", o)
	_, _, err := m.Load(name)
	if err != nil {
		err = cssCompiler(m, name, assets, o)
		if err != nil {
			return nil, err
		}
	}
	return CssParser(m, []string{name}, o)
}

func init() {
	registerCompiler(CssCompiler, CodeTypeCss, true)
}
