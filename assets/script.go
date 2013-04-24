package assets

type ScriptAsset struct {
	*CommonAsset
}

func (c *ScriptAsset) Position() Position {
	return Bottom
}

func ScriptParser(m Manager, names []string, options Options) ([]Asset, error) {
	common, err := ParseCommonAssets(m, names, options)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, len(common))
	for ii, v := range common {
		v.TagName = "script"
		v.MustClose = true
		v.Attributes = Attributes{"type": "text/javascript", "src": m.URL(v.Name)}
		assets[ii] = &ScriptAsset{
			CommonAsset: v,
		}
	}
	return assets, nil
}

func init() {
	Register("script", ScriptParser)
	Register("scripts", ScriptParser)
}
