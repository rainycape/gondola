package assets

import (
	"fmt"
	"gondola/files"
)

type CssAsset struct {
	*CommonAsset
}

func (c *CssAsset) Position() Position {
	return Top
}

func CssParser(dir string, names []string, options map[string]string) ([]Asset, error) {
	common, err := ParseCommonAssets(dir, names, options)
	if err != nil {
		return nil, err
	}
	attrs := Attributes{"rel": "stylesheet"}
	for k, v := range options {
		if k == "media" {
			attrs[k] = v
		} else {
			return nil, fmt.Errorf("Unknown CSS option %q", k)
		}
	}
	assets := make([]Asset, len(common))
	for ii, v := range common {
		v.TagName = "link"
		v.Attributes = make(Attributes, len(attrs)+1)
		for ak, av := range attrs {
			v.Attributes[ak] = av
		}
		v.Attributes["href"] = files.StaticFileUrl(dir, v.Name)
		assets[ii] = &CssAsset{
			CommonAsset: v,
		}
	}
	return assets, nil
}

func init() {
	Register("css", CssParser)
	Register("style", CssParser)
	Register("styles", CssParser)
}
