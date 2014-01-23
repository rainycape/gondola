package bootstrap

import (
	"fmt"
	"gnd.la/template/assets"
	"gnd.la/util/semver"
)

const (
	bootstrapCSSFmt              = "//netdna.bootstrapcdn.com/bootstrap/%s/css/bootstrap.min.css"
	bootstrapCSSNoIconsFmt       = "//netdna.bootstrapcdn.com/bootstrap/%s/css/bootstrap.no-icons.min.css"
	bootstrapCSSNoIconsLegacyFmt = "//netdna.bootstrapcdn.com/bootstrap/%s/css/bootstrap-combined.no-icons.min.css"
	bootstrapCSSThemeFmt         = "http://netdna.bootstrapcdn.com/bootstrap/%s/css/bootstrap-theme.min.css"
	bootstrapJSFmt               = "//netdna.bootstrapcdn.com/bootstrap/%s/js/bootstrap.min.js"
	fontAwesomeFmt               = "//netdna.bootstrapcdn.com/font-awesome/%s/css/font-awesome.min.css"
)

func bootstrapParser(m *assets.Manager, names []string, options assets.Options) ([]*assets.Asset, error) {
	if len(names) > 1 {
		return nil, fmt.Errorf("invalid bootstrap declaration \"%s\": must include only a version number", names)
	}
	bsV := names[0]
	bsVersion, err := semver.Parse(bsV)
	if err != nil || bsVersion.PreRelease != "" || bsVersion.Build != "" {
		return nil, fmt.Errorf("invalid bootstrap version %q", bsV)
	}
	if bsVersion.Major != 2 && bsVersion.Major != 3 {
		return nil, fmt.Errorf("only bootstrap versions 2.x and 3.x are supported")
	}
	var as []*assets.Asset
	if options.BoolOpt("fontawesome") {
		faV := options.StringOpt("fontawesome")
		if faV == "" {
			return nil, fmt.Errorf("please, specify a font awesome version")
		}
		faVersion, err := semver.Parse(faV)
		if err != nil || faVersion.PreRelease != "" || faVersion.Build != "" {
			return nil, fmt.Errorf("invalid font awesome version %q", faV)
		}
		if faVersion.Major != 3 && faVersion.Major != 4 {
			return nil, fmt.Errorf("only font awesome versions 3.x and 4.x are supported")
		}
		format := bootstrapCSSFmt
		if bsVersion.Major == 2 {
			format = bootstrapCSSNoIconsLegacyFmt
		} else if faVersion.Major == 3 {
			if bsVersion.Major >= 3 && (bsVersion.Minor > 0 || bsVersion.Patch > 0) {
				return nil, fmt.Errorf("can't use bootstrap > 3.0.0 with font awesome 3 (bootstrapcdn does not provide the files)")
			} else {
				format = bootstrapCSSNoIconsFmt
			}
		}
		as = append(as, assets.CSS(fmt.Sprintf(format, bsV)))
		as = append(as, assets.CSS(fmt.Sprintf(fontAwesomeFmt, faV)))
	} else {
		as = append(as, assets.CSS(fmt.Sprintf(bootstrapCSSFmt, bsV)))
	}
	if options.BoolOpt("theme") && bsVersion.Major == 3 {
		as = append(as, assets.CSS(fmt.Sprintf(bootstrapCSSThemeFmt, bsV)))
	}
	// Required for IE8 support
	html5Shiv := assets.Script("https://oss.maxcdn.com/libs/html5shiv/3.7.0/html5shiv.js")
	respondJs := assets.Script("https://oss.maxcdn.com/libs/respond.js/1.3.0/respond.min.js")
	cond := &assets.Condition{Comparison: assets.ComparisonLessThan, Version: 9}
	html5Shiv.Condition = cond
	respondJs.Condition = cond
	as = append(as, html5Shiv, respondJs)
	if !options.BoolOpt("nojs") {
		as = append(as, assets.Script(fmt.Sprintf(bootstrapJSFmt, bsV)))
	}
	return as, nil
}

func fontAwesomeParser(m *assets.Manager, version string, opts assets.Options) ([]*assets.Asset, error) {
	return []*assets.Asset{assets.CSS(fmt.Sprintf(fontAwesomeFmt, version))}, nil
}

func init() {
	assets.Register("bootstrap", bootstrapParser)
	assets.Register("fontawesome", assets.SingleParser(fontAwesomeParser))
}
