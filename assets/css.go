package assets

type cssAsset struct {
	*CommonAsset
	attributes Attributes
}

func (c *cssAsset) Tag() string {
	return "link"
}

func (c *cssAsset) Closed() bool {
	return false
}

func (c *cssAsset) Position() Position {
	return Top
}

func (c *cssAsset) Attributes() Attributes {
	return c.attributes
}

func (c *cssAsset) HTML() string {
	return ""
}

func (c *cssAsset) CodeType() int {
	return CodeTypeCss
}

func cssParser(m Manager, names []string, options Options) ([]Asset, error) {
	common, err := ParseCommonAssets(m, names, options)
	if err != nil {
		return nil, err
	}
	attrs := Attributes{"rel": "stylesheet", "type": "text/css"}
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
		assets[ii] = &cssAsset{
			CommonAsset: v,
			attributes:  attributes,
		}
	}
	return assets, nil
}

func init() {
	Register("css", cssParser)
	Register("style", cssParser)
	Register("styles", cssParser)
}
