package assets

import (
	"gondola/files"
)

type ScriptAsset struct {
	*CommonAsset
}

func (c *ScriptAsset) Position() Position {
	return Bottom
}

func ScriptParser(dir string, names []string, options map[string]string) ([]Asset, error) {
	common, err := ParseCommonAssets(dir, names, options)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, len(common))
	for ii, v := range common {
		v.TagName = "script"
		v.MustClose = true
		v.Attributes = Attributes{"type": "text/javascript", "src": files.StaticFileUrl(dir, v.Name)}
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
