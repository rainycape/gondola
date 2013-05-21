package assets

import (
	"gondola/log"
)

type ScriptAsset struct {
	*CommonAsset
	position   Position
	attributes Attributes
}

func (s *ScriptAsset) Tag() string {
	return "script"
}

func (s *ScriptAsset) Closed() bool {
	return true
}

func (s *ScriptAsset) Position() Position {
	return s.position
}

func (s *ScriptAsset) Attributes() Attributes {
	return s.attributes
}

func (s *ScriptAsset) HTML() string {
	return ""
}

func (s *ScriptAsset) CodeType() int {
	return CodeTypeJavascript
}

func ScriptParser(m Manager, names []string, options Options) ([]Asset, error) {
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
		assets[ii] = &ScriptAsset{
			CommonAsset: v,
			position:    position,
			attributes:  attrs,
		}
	}
	return assets, nil
}

func init() {
	Register("script", ScriptParser)
	Register("scripts", ScriptParser)
}
