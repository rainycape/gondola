package assets

func CSS(name string) *Asset {
	return &Asset{
		Name:     name,
		Position: Top,
		Type:     TypeCSS,
	}
}

func cssParser(m *Manager, names []string, options Options) ([]*Asset, error) {
	var media string
	for k, v := range options {
		if k == "media" {
			media = v
		}
	}
	pos := Top
	if options.Bottom() {
		pos = Bottom
	}
	assets := make([]*Asset, len(names))
	for ii, v := range names {
		asset := CSS(v)
		if media != "" {
			asset.Attributes = Attributes{"media": media}
		}
		asset.Position = pos
		assets[ii] = asset
	}
	return assets, nil
}

func init() {
	Register("css", cssParser)
	Register("style", cssParser)
	Register("styles", cssParser)
}
