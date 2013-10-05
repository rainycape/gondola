package assets

import (
	"gnd.la/log"
)

type scriptAsset struct {
	*CommonAsset
	position   Position
	attributes Attributes
}

func (s *scriptAsset) Tag() string {
	return "script"
}

func (s *scriptAsset) Closed() bool {
	return true
}

func (s *scriptAsset) Position() Position {
	return s.position
}

func (s *scriptAsset) Attributes() Attributes {
	return s.attributes
}

func (s *scriptAsset) HTML() string {
	return ""
}

func (s *scriptAsset) CodeType() int {
	return CodeTypeJavascript
}

func scriptParser(m Manager, names []string, options Options) ([]Asset, error) {
	common, err := ParseCommonAssets(m, names, options)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, len(common))
	cdn := options.BoolOpt("cdn", m)
	position := Bottom
	if options.BoolOpt("top", m) {
		position = Top
	}
	async := options.BoolOpt("async", m)
	for ii, v := range common {
		var src string
		name := v.Name()
		if cdn {
			var err error
			src, err = Cdn(name)
			if err != nil {
				log.Warning("Error finding CDN URL for %q: %s. Using original.", name, err)
			} else {
				log.Debugf("Using CDN URL %q for %q", src, name)
			}
		}
		if src == "" {
			src = m.URL(name)
		}
		attrs := Attributes{"type": "text/javascript", "src": src}
		if async {
			attrs["async"] = "async"
		}
		assets[ii] = &scriptAsset{
			CommonAsset: v,
			position:    position,
			attributes:  attrs,
		}
	}
	return assets, nil
}

func init() {
	Register("script", scriptParser)
	Register("scripts", scriptParser)
}
