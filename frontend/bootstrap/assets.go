package bootstrap

import (
	"fmt"

	"gnd.la/template/assets"

	"gopkgs.com/semver.v1"
)

const (
	bootstrapCSSFmt      = "//netdna.bootstrapcdn.com/bootstrap/%s/css/bootstrap.min.css"
	bootstrapCSSThemeFmt = "//netdna.bootstrapcdn.com/bootstrap/%s/css/bootstrap-theme.min.css"
	bootstrapJSFmt       = "//netdna.bootstrapcdn.com/bootstrap/%s/js/bootstrap.min.js"
)

func bootstrapParser(m *assets.Manager, version string, options assets.Options) ([]*assets.Asset, error) {
	bsVersion, err := semver.Parse(version)
	if err != nil || bsVersion.Major != 3 || bsVersion.PreRelease != "" || bsVersion.Build != "" {
		return nil, fmt.Errorf("invalid bootstrap version %q, must be in 3.x.y form", version)
	}
	as := []*assets.Asset{
		assets.CSS(fmt.Sprintf(bootstrapCSSFmt, version)),
	}
	if options.BoolOpt("theme") {
		as = append(as, assets.CSS(fmt.Sprintf(bootstrapCSSThemeFmt, version)))
	}
	// Required for IE8 support
	html5Shiv := assets.Script("https://oss.maxcdn.com/libs/html5shiv/3.7.0/html5shiv.js")
	respondJs := assets.Script("https://oss.maxcdn.com/libs/respond.js/1.3.0/respond.min.js")
	cond := &assets.Condition{Comparison: assets.ComparisonLessThan, Version: 9}
	html5Shiv.Condition = cond
	respondJs.Condition = cond
	html5Shiv.Position = assets.Top
	respondJs.Position = assets.Top
	as = append(as, html5Shiv, respondJs)
	if !options.BoolOpt("nojs") {
		as = append(as, assets.Script(fmt.Sprintf(bootstrapJSFmt, version)))
	}
	return as, nil
}

func init() {
	assets.Register("bootstrap", assets.SingleParser(bootstrapParser))
}
