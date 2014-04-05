package assets

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"strings"
)

type CdnInfo struct {
	Pattern  *regexp.Regexp
	Repl     string
	Fallback string
}

var CdnInfos = []*CdnInfo{
	{Pattern: regexp.MustCompile("angular-([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/angularjs/$1/angular.min.js"},
	{Pattern: regexp.MustCompile("CFInstall-([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/chrome-frame/$1/CFInstall.min.js"},
	{Pattern: regexp.MustCompile("dojo-([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/dojo/$1/dojo/dojo.js"},
	{Pattern: regexp.MustCompile("ext-core-([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/ext-core/$1/ext-core.js"},
	{Pattern: regexp.MustCompile("jquery-([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/jquery/$1/jquery.min.js", Fallback: "window.jQuery"},
	{Pattern: regexp.MustCompile("jquery-ui-([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/jqueryui/$1/jquery-ui.min.js"},
	{Pattern: regexp.MustCompile("mootools-(:?core-)?([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/mootools/$1/mootools-yui-compressed.js"},
	{Pattern: regexp.MustCompile("prototype-([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/prototype/$1/prototype.js"},
	{Pattern: regexp.MustCompile("scriptaculous-([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/scriptaculous/$1/scriptaculous.js"},
	{Pattern: regexp.MustCompile("swfobject-([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/swfobject/$1/swfobject.js"},
	{Pattern: regexp.MustCompile("webfont-([\\d\\.]+\\d)"), Repl: "//ajax.googleapis.com/ajax/libs/webfont/$1/webfont.js"},
}

func CdnAssets(m *Manager, asset *Asset) ([]*Asset, error) {
	name, fallback, err := Cdn(asset.Name)
	if err != nil {
		return nil, err
	}
	acdn := *asset
	acdn.Name = name
	assets := []*Asset{&acdn}
	if err := appendScriptFallback(m, asset, &assets, fallback); err != nil {
		return nil, err
	}
	return assets, nil
}

func Cdn(name string) (string, string, error) {
	base := filepath.Base(name)
	for _, v := range CdnInfos {
		m := v.Pattern.FindStringSubmatchIndex(base)
		if m != nil {
			var dst []byte
			return string(v.Pattern.ExpandString(dst, v.Repl, base, m)), v.Fallback, nil
		}
	}
	return "", "", fmt.Errorf("could not find CDN URL for %q", name)
}

func cdnScriptParser(k, orig string) SingleAssetParser {
	return func(m *Manager, name string, options Options) ([]*Asset, error) {
		asset := orig + name
		src, fallback, err := Cdn(asset)
		if err != nil {
			return nil, err
		}
		position := Bottom
		if options.Top() {
			position = Top
		}
		script := Script(src)
		script.Position = position
		if options.Async() {
			script.Attributes = Attributes{"async": "async"}
		}
		assets := []*Asset{script}
		if err := appendScriptFallback(m, script, &assets, fallback); err != nil {
			return nil, err
		}
		return assets, nil
	}
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
	for _, v := range CdnInfos {
		re, _ := syntax.Parse(v.Pattern.String(), syntax.Perl)
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
		Register(key, SingleParser(cdnScriptParser(key, orig)))
	}
}
