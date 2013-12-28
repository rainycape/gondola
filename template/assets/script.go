package assets

func Script(name string, src string) *Asset {
	return &Asset{
		Name:       name,
		Position:   Bottom,
		CodeType:   CodeTypeJavascript,
		Tag:        "script",
		Closed:     true,
		Attributes: Attributes{"type": "text/javascript", "src": src},
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
		asset := Script(v, m.URL(v))
		if async {
			asset.Attributes["async"] = "async"
			asset.Position = position
		}
		assets[ii] = asset
		/*var src string
		name := v.Name
		if cdn {
			var err error
			src, err = Cdn(name)
			if err != nil {
				log.Warning("Error finding CDN URL for %q: %s. Using original.", name, err)
			} else {
				log.Debugf("Using CDN URL %q for %q", src, name)
			}
		}
		if src == "" {
			src = m.URL(name)
		}*/
	}
	return assets, nil
}

func init() {
	Register("script", scriptParser)
	Register("scripts", scriptParser)
}
