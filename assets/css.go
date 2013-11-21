package assets

type Css struct {
	Common
	Media      string
	Href       string
	attributes Attributes
}

func (c *Css) AssetTag() string {
	return "link"
}

func (c *Css) AssetClosed() bool {
	return false
}

func (c *Css) AssetPosition() Position {
	return Top
}

func (c *Css) AssetAttributes() Attributes {
	if c.attributes == nil {
		c.attributes = Attributes{"rel": "stylesheet", "type": "text/css"}
		if c.Media != "" {
			c.attributes["media"] = c.Media
		}
		if c.Href != "" {
			c.attributes["href"] = c.Href
		}
	}
	return c.attributes
}

func (c *Css) AssetHTML() string {
	return ""
}

func (c *Css) CodeType() CodeType {
	return CodeTypeCss
}

func cssParser(m Manager, names []string, options Options) ([]Asset, error) {
	common, err := ParseCommon(m, names, CodeTypeCss, options)
	if err != nil {
		return nil, err
	}
	var media string
	for k, v := range options {
		if k == "media" {
			media = v
		}
	}
	assets := make([]Asset, len(common))
	for ii, v := range common {
		assets[ii] = &Css{
			Common: *v,
			Media:  media,
			Href:   m.URL(v.Name),
		}
	}
	return assets, nil
}

func init() {
	Register("css", cssParser)
	Register("style", cssParser)
	Register("styles", cssParser)
}
