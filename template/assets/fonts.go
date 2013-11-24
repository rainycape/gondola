package assets

import (
	"net/url"
	"strings"
)

func googleFontsParser(m Manager, names []string, options Options) ([]Asset, error) {
	var families []string
	for _, v := range names {
		// Format is "font name:size1:size2:..., but Google
		// Fonts expects "font name:size1,size2,..."
		colon := strings.IndexByte(v, ':')
		if colon >= 0 {
			v = v[:colon+1] + strings.Replace(v[colon+2:], ":", ",", -1)
		}
		families = append(families, v)
	}
	values := url.Values{}
	values.Add("family", strings.Join(families, "|"))
	subset := options.StringOpt("subset", m)
	if subset != "" {
		values.Add("subset", subset)
	}
	return []Asset{
		&Css{
			Common: Common{
				Manager: m,
				Name:    "google-fonts.css",
			},
			Href: "//fonts.googleapis.com/css?" + values.Encode(),
		},
	}, nil
}

func init() {
	Register("google-fonts", googleFontsParser)
}
