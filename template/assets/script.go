package assets

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"gnd.la/crypto/hashutil"
	"gnd.la/log"
	"gnd.la/net/urlutil"
)

func Script(name string) *Asset {
	return &Asset{
		Name:     name,
		Position: Bottom,
		Type:     TypeJavascript,
	}
}

// Create a local fallback for the given script, downloading it if
// necessary
func scriptFallback(m *Manager, script *Asset, fallback string) (*Asset, error) {
	fallbackName := script.Name
	if !m.Has(fallbackName) {
		var scriptURL string
		if urlutil.IsURL(fallbackName) {
			scriptURL = fallbackName
			fallbackName = "asset.gen." + hashutil.Adler32(fallbackName) + "." + path.Base(fallbackName)
		} else {
			cdn, _, err := Cdn(fallbackName)
			if err != nil {
				return nil, err
			}
			scriptURL = cdn
		}
		if !m.Has(fallbackName) {
			u, err := url.Parse(scriptURL)
			if err != nil {
				return nil, err
			}
			if u.Scheme == "" {
				u.Scheme = "http"
			}
			log.Debugf("fetching local fallback for %s to %s", u, fallbackName)
			resp, err := http.Get(u.String())
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			w, err := m.Create(fallbackName, true)
			if err != nil {
				return nil, err
			}
			defer w.Close()
			if _, err := io.Copy(w, resp.Body); err != nil {
				return nil, err
			}
		}
	}
	return &Asset{
		Name:     fallbackName,
		Position: Bottom,
		Type:     TypeOther,
		HTML:     fmt.Sprintf("<script>%s || document.write('<scr'+'ipt src=\"%s\"><\\/scr'+'ipt>')</script>", fallback, m.URL(fallbackName)),
	}, nil
}

func appendScriptFallback(m *Manager, script *Asset, assets *[]*Asset, fallback string) error {
	if fallback != "" {
		if _, ok := script.Attributes["async"]; ok {
			log.Debugf("skipping fallback for async script %s", script.Name)
			return nil
		}
		fb, err := scriptFallback(m, script, fallback)
		if err != nil {
			return err
		}
		*assets = append(*assets, fb)
	}
	return nil
}

func scriptParser(m *Manager, names []string, options Options) ([]*Asset, error) {
	assets := make([]*Asset, len(names))
	position := Bottom
	if options.Top() {
		position = Top
	}
	async := options.Async()
	for ii, v := range names {
		asset := Script(v)
		asset.Position = position
		if async {
			asset.Attributes = Attributes{"async": "async"}
		}
		assets[ii] = asset
	}
	return assets, nil
}

func init() {
	Register("script", scriptParser)
	Register("scripts", scriptParser)
}
