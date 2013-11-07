package assets

import (
	"gnd.la/log"
)

type Script struct {
	Common
	Position   Position
	Async      bool
	Src        string
	Script     string
	attributes Attributes
}

func (s *Script) AssetTag() string {
	return "script"
}

func (s *Script) AssetClosed() bool {
	return true
}

func (s *Script) AssetPosition() Position {
	return s.Position
}

func (s *Script) AssetAttributes() Attributes {
	if s.attributes == nil {
		s.attributes = Attributes{"type": "text/javascript", "src": s.Src}
		if s.Async {
			s.attributes["async"] = "async"
		}
	}
	return s.attributes
}

func (s *Script) AssetHTML() string {
	return s.Script
}

func (s *Script) CodeType() int {
	return CodeTypeJavascript
}

func scriptParser(m Manager, names []string, options Options) ([]Asset, error) {
	common, err := ParseCommon(m, names, options)
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
		name := v.Name
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
		assets[ii] = &Script{
			Common:   *v,
			Position: position,
			Async:    async,
			Src:      src,
		}
	}
	return assets, nil
}

func init() {
	Register("script", scriptParser)
	Register("scripts", scriptParser)
}
