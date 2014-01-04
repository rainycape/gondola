package assets

func Script(name string) *Asset {
	return &Asset{
		Name:     name,
		Position: Bottom,
		Type:     TypeJavascript,
	}
}

func scriptParser(m *Manager, names []string, options Options) ([]*Asset, error) {
	assets := make([]*Asset, len(names))
	position := Bottom
	if options.Top() {
		position = Top
	}
	async := options.Async()
	for ii, v := range names {
		asset := Script(v)
		asset.Position = position
		if async {
			asset.Attributes = Attributes{"async": "async"}
		}
		assets[ii] = asset
	}
	return assets, nil
}

func init() {
	Register("script", scriptParser)
	Register("scripts", scriptParser)
}
