package assets

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"strings"
)

type CdnMap map[*regexp.Regexp]string

var cdnMap = CdnMap{
	regexp.MustCompile("angular-([\\d\\.]+\\d)"):            "//ajax.googleapis.com/ajax/libs/angularjs/$1/angular.min.js",
	regexp.MustCompile("CFInstall-([\\d\\.]+\\d)"):          "//ajax.googleapis.com/ajax/libs/chrome-frame/$1/CFInstall.min.js",
	regexp.MustCompile("dojo-([\\d\\.]+\\d)"):               "//ajax.googleapis.com/ajax/libs/dojo/$1/dojo/dojo.js",
	regexp.MustCompile("ext-core-([\\d\\.]+\\d)"):           "//ajax.googleapis.com/ajax/libs/ext-core/$1/ext-core.js",
	regexp.MustCompile("jquery-([\\d\\.]+\\d)"):             "//ajax.googleapis.com/ajax/libs/jquery/$1/jquery.min.js",
	regexp.MustCompile("jquery-ui-([\\d\\.]+\\d)"):          "//ajax.googleapis.com/ajax/libs/jquery/$1/jquery-ui.min.js",
	regexp.MustCompile("mootools-(:?core-)?([\\d\\.]+\\d)"): "//ajax.googleapis.com/ajax/libs/mootools/$1/mootools-yui-compressed.js",
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
	return "", fmt.Errorf("could not find CDN URL for %q", name)
}

func cdnScriptParser(k, orig string) func(m Manager, names []string, options Options) ([]Asset, error) {
	fn := func(m Manager, names []string, options Options) ([]Asset, error) {
		common, err := ParseCommon(m, names, options)
		if err != nil {
			return nil, err
		}
		name := orig + names[0]
		src, err := Cdn(name)
		if err != nil {
			return nil, err
		}
		position := Bottom
		if options.BoolOpt("top", m) {
			position = Top
		}
		return []Asset{
			&Script{
				Common:   *common[0],
				Position: position,
				Async:    options.BoolOpt("async", m),
				Src:      src,
			},
		}, nil
	}
	return singleParser(fn)
}

func walk(r *syntax.Regexp, f func(*syntax.Regexp) bool) bool {
	stop := f(r)
	if !stop {
		for _, v := range r.Sub {
			if walk(v, f) {
				stop = true
				break
			}
		}
	}
	return stop
}

func init() {
	for k := range cdnMap {
		re, _ := syntax.Parse(k.String(), syntax.Perl)
		var buf bytes.Buffer
		walk(re, func(r *syntax.Regexp) bool {
			if r.Op == syntax.OpLiteral {
				buf.WriteString(string(r.Rune))
				return false
			}
			if r.Op == syntax.OpConcat {
				return false
			}
			return true
		})
		orig := buf.String()
		key := strings.Trim(orig, " -")
		Register(key, cdnScriptParser(key, orig))
	}
}
