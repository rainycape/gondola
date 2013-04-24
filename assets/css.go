package assets

type CssAsset struct {
	*CommonAsset
	attributes Attributes
}

func (c *CssAsset) Tag() string {
	return "link"
}

func (c *CssAsset) Closed() bool {
	return false
}

func (c *CssAsset) Position() Position {
	return Top
}

func (c *CssAsset) Attributes() Attributes {
	return c.attributes
}

func (c *CssAsset) HTML() string {
	return ""
}

func (c *CssAsset) CodeType() int {
	return CodeTypeCss
}

func CssParser(m Manager, names []string, options Options) ([]Asset, error) {
	common, err := ParseCommonAssets(m, names, options)
	if err != nil {
		return nil, err
	}
	attrs := Attributes{"rel": "stylesheet"}
	for k, v := range options {
		if k == "media" {
			attrs[k] = v
		}
	}
	assets := make([]Asset, len(common))
	for ii, v := range common {
		attributes := make(Attributes, len(attrs)+1)
		for ak, av := range attrs {
			attributes[ak] = av
		}
		attributes["href"] = m.URL(v.Name())
		assets[ii] = &CssAsset{
			CommonAsset: v,
			attributes:  attributes,
		}
	}
	return assets, nil
}

func init() {
	Register("css", CssParser)
	Register("style", CssParser)
	Register("styles", CssParser)
}
