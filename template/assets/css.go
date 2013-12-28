package assets

func CSS(name string, href string) *Asset {
	return &Asset{
		Name:       name,
		Position:   Top,
		CodeType:   CodeTypeCss,
		Tag:        "link",
		Attributes: Attributes{"rel": "stylesheet", "type": "text/css", "href": href},
	}
}

func cssParser(m *Manager, names []string, options Options) ([]*Asset, error) {
	var media string
	for k, v := range options {
		if k == "media" {
			media = v
		}
	}
	assets := make([]*Asset, len(names))
	for ii, v := range names {
		asset := CSS(v, m.URL(v))
		if media != "" {
			asset.Attributes["media"] = media
		}
		assets[ii] = asset
	}
	return assets, nil
}

func init() {
	Register("css", cssParser)
	Register("style", cssParser)
	Register("styles", cssParser)
}
