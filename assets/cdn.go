package assets

import (
	"fmt"
	"path/filepath"
	"regexp"
)

type CdnMap map[*regexp.Regexp]string

var cdnMap = CdnMap{
	regexp.MustCompile("angular-([\\d\\.]+\\d)"):            "//ajax.googleapis.com/ajax/libs/angularjs/1.0.6/angular.min.js",
	regexp.MustCompile("CFInstall-([\\d\\.]+\\d)"):          "//ajax.googleapis.com/ajax/libs/chrome-frame/$1/CFInstall.min.js",
	regexp.MustCompile("dojo-([\\d\\.]+\\d)"):               "//ajax.googleapis.com/ajax/libs/dojo/$1/dojo/dojo.js",
	regexp.MustCompile("ext-core-([\\d\\.]+\\d)"):           "//ajax.googleapis.com/ajax/libs/ext-core/$1/ext-core.js",
	regexp.MustCompile("jquery-([\\d\\.]+\\d)"):             "//ajax.googleapis.com/ajax/libs/jquery/$1/jquery.min.js",
	regexp.MustCompile("jquery-ui-([\\d\\.]+\\d)"):          "//ajax.googleapis.com/ajax/libs/jquery/$1/jquery-ui.min.js",
	regexp.MustCompile("mootools(:?-core)?-([\\d\\.]+\\d)"): "//ajax.googleapis.com/ajax/libs/mootools/$1/mootools-yui-compressed.js",
	regexp.MustCompile("prototype-([\\d\\.]+\\d)"):          "//ajax.googleapis.com/ajax/libs/prototype/$1/prototype.js",
	regexp.MustCompile("scriptaculous-([\\d\\.]+\\d)"):      "//ajax.googleapis.com/ajax/libs/scriptaculous/$1/scriptaculous.js",
	regexp.MustCompile("swfobject-([\\d\\.]+\\d)"):          "//ajax.googleapis.com/ajax/libs/swfobject/$1/swfobject.js",
	regexp.MustCompile("webfont-([\\d\\.]+\\d)"):            "//ajax.googleapis.com/ajax/libs/webfont/$1/webfont.js",
}

func Cdn(name string) (string, error) {
	base := filepath.Base(name)
	for k, v := range cdnMap {
		m := k.FindStringSubmatchIndex(base)
		if m != nil {
			var dst []byte
			return string(k.ExpandString(dst, v, base, m)), nil
		}
	}
	return "", fmt.Errorf("Could not find CDN URL for %q", name)
}
