package assets

type ScriptAsset struct {
	*CommonAsset
	attributes Attributes
}

func (s *ScriptAsset) Tag() string {
	return "script"
}

func (s *ScriptAsset) Closed() bool {
	return true
}

func (s *ScriptAsset) Position() Position {
	return Bottom
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
	for ii, v := range common {
		assets[ii] = &ScriptAsset{
			CommonAsset: v,
			attributes:  Attributes{"type": "text/javascript", "src": m.URL(v.Name())},
		}
	}
	return assets, nil
}

func init() {
	Register("script", ScriptParser)
	Register("scripts", ScriptParser)
}
