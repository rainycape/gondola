package bootstrap

import (
	"fmt"
	"gnd.la/assets"
	"strings"
)

const (
	bootstrapCssFmt              = "//netdna.bootstrapcdn.com/bootstrap/%s/css/bootstrap.min.css"
	bootstrapCssNoIconsFmt       = "//netdna.bootstrapcdn.com/bootstrap/%s/css/bootstrap.no-icons.min.css"
	bootstrapCssNoIconsLegacyFmt = "//netdna.bootstrapcdn.com/bootstrap/%s/css/bootstrap-combined.no-icons.min.css"
	bootstrapJsFmt               = "//netdna.bootstrapcdn.com/bootstrap/%s/js/bootstrap.min.js"
	fontAwesomeFmt               = "//netdna.bootstrapcdn.com/font-awesome/%s/css/font-awesome.min.css"
)

func bootstrapParser(m assets.Manager, names []string, options assets.Options) ([]assets.Asset, error) {
	if len(names) > 1 {
		return nil, fmt.Errorf("invalid bootstrap declaration \"%s\": must include only a version number", names)
	}
	version := names[0]
	if !strings.HasPrefix(version, "3.") && !strings.HasPrefix(version, "2.") {
		return nil, fmt.Errorf("invalid bootstrap version %q", version)
	}
	var as []assets.Asset
	if options.BoolOpt("fontawesome", m) {
		format := bootstrapCssNoIconsFmt
		if strings.HasPrefix(version, "2.") {
			format = bootstrapCssNoIconsLegacyFmt
		}
		as = append(as, &assets.Css{
			Common: assets.Common{
				Manager: m,
				Name:    fmt.Sprintf("bootstrap-noicons-%s.css", version),
			},
			Href: fmt.Sprintf(format, version),
		})
		faVersion := "3.2.1"
		if v := options.StringOpt("fontawesome", m); v != "" {
			faVersion = v
		}
		as = append(as, &assets.Css{
			Common: assets.Common{
				Manager: m,
				Name:    fmt.Sprintf("fontawesome-%s.css", faVersion),
			},
			Href: fmt.Sprintf(fontAwesomeFmt, faVersion),
		})
	} else {
		as = append(as, &assets.Css{
			Common: assets.Common{
				Manager: m,
				Name:    fmt.Sprintf("bootstrap-%s.css", version),
			},
			Href: fmt.Sprintf(bootstrapCssFmt, version),
		})
	}
	if !options.BoolOpt("nojs", m) {
		as = append(as, &assets.Script{
			Common: assets.Common{
				Manager: m,
				Name:    fmt.Sprintf("bootstrap-%s.js", version),
			},
			Src: fmt.Sprintf(bootstrapJsFmt, version),
		})
	}
	return as, nil
}

func init() {
	assets.Register("bootstrap", bootstrapParser)
}
