package assets

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"gnd.la/log"
	"hash/fnv"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
)

type CodeType int

const (
	CodeTypeNone CodeType = iota
	CodeTypeCss
	CodeTypeJavascript
)

func (c CodeType) String() string {
	switch c {
	case CodeTypeCss:
		return "CSS"
	case CodeTypeJavascript:
		return "Javascript"
	}
	return "unknown CodeType"
}

func (c CodeType) Ext() string {
	switch c {
	case CodeTypeCss:
		return "css"
	case CodeTypeJavascript:
		return "js"
	}
	return ""
}

var (
	errNoAssets = errors.New("no assets to bundle")
	urlRe       = regexp.MustCompile("i?url\\s*?\\((.*?)\\)")
)

type bundledAssets []*Asset

func (a bundledAssets) Names() []string {
	var names []string
	for _, v := range a {
		names = append(names, v.Name)
	}
	return names
}

func (a bundledAssets) BundleName(m *Manager, ext string, o Options) (string, error) {
	if len(a) == 0 {
		return "", nil
	}
	h := fnv.New32a()
	for _, v := range a {
		code, err := v.Code(m)
		if err != nil {
			return "", err
		}
		io.WriteString(h, code)
	}
	io.WriteString(h, o.String())
	sum := hex.EncodeToString(h.Sum(nil))
	name := a[0].Name
	if ext == "" {
		ext = path.Ext(name)
	} else {
		ext = "." + ext
	}
	return path.Join(path.Dir(name), "bundle.gen."+sum+ext), nil
}

func Bundle(m *Manager, assets []*Asset, opts Options) (*Asset, error) {
	if len(assets) == 0 {
		return nil, errNoAssets
	}
	codeType := CodeType(-1)
	for _, v := range assets {
		if v.CodeType == CodeTypeNone {
			return nil, fmt.Errorf("asset %q does not specify a CodeType and can't be bundled", v.Name)
		}
		if codeType < 0 {
			codeType = v.CodeType
		} else if codeType != v.CodeType {
			return nil, fmt.Errorf("asset %q has different code type %s (first asset is of type %s)", v.Name, v.CodeType, codeType)
		}
	}
	bundler := bundlers[codeType]
	if bundler == nil {
		return nil, fmt.Errorf("no bundler for %s", codeType)
	}
	// Prepare the code, changing relative paths if required
	name, err := bundledAssets(assets).BundleName(m, codeType.Ext(), opts)
	if err != nil {
		return nil, err
	}
	dir := path.Dir(name)
	names := bundledAssets(assets).Names()
	// Check if the code has been already bundled
	if f, _, err := m.Load(name); err == nil {
		f.Close()
		log.Debugf("%s already bundled into %s and up to date", names, name)
	} else {
		log.Debugf("bundling %v", names)
		var code []string
		for _, v := range assets {
			c, err := v.Code(m)
			if err != nil {
				return nil, fmt.Errorf("error getting code for asset %q: %s", v.Name, err)
			}
			if vd := path.Dir(v.Name); vd != dir {
				if codeType == CodeTypeCss {
					log.Debugf("asset %q will move from %v to %v, rewriting relative paths...", v.Name, vd, dir)
					c = replaceRelativePaths(c, vd, dir)
				} else {
					log.Warningf("asset %q will move from %v to %v, relative paths might not work", v.Name, vd, dir)
				}
			}
			code = append(code, c)
		}
		// Bundle to a buf first. We don't want to create
		// the file if the bundling fails.
		var buf bytes.Buffer
		reader := strings.NewReader(strings.Join(code, "\n\n"))
		if err := bundler.Bundle(&buf, reader, m, opts); err != nil {
			return nil, err
		}
		w, err := m.Create(name, true)
		if err == nil {
			s := makeLinksCacheable(m, dir, buf.Bytes())
			if _, err := io.Copy(w, strings.NewReader(s)); err != nil {
				w.Close()
				return nil, err
			}
			if err := w.Close(); err != nil {
				return nil, err
			}
		} else {
			// If the file exists, is up to date
			if !os.IsExist(err) {
				return nil, err
			}
		}
	}
	return bundler.Asset(name, m, opts)
}

func makeLinksCacheable(m *Manager, dir string, b []byte) string {
	css := string(b)
	return replaceCssUrls(css, func(s string) string {
		var suffix string
		if sep := strings.IndexAny(s, "?#"); sep >= 0 {
			suffix = s[sep:]
			s = s[:sep]
		}
		p := path.Join(dir, s)
		base := m.URL(p)
		if strings.Contains(base, "?") && suffix != "" && suffix[0] == '?' {
			suffix = "&" + suffix[1:]
		}
		repl := base + suffix
		return repl
	})
}

func replaceRelativePaths(code string, dir string, final string) string {
	count := strings.Count(final, "/") + 1
	return replaceCssUrls(code, func(s string) string {
		old := path.Join(dir, s)
		return strings.Repeat("../", count) + old
	})
}

func replaceCssUrls(code string, f func(string) string) string {
	return urlRe.ReplaceAllStringFunc(code, func(s string) string {
		r := urlRe.FindStringSubmatch(s)
		p := r[1]
		quote := ""
		if len(p) > 0 && (p[0] == '\'' || p[0] == '"') {
			quote = string(p[0])
			p = p[1 : len(p)-1]
		}
		if !urlIsRelative(p) {
			return s
		}
		repl := f(p)
		if repl == p {
			return s
		}
		return fmt.Sprintf("url(%s%s%s)", quote, repl, quote)
	})
}

func urlIsRelative(u string) bool {
	return !strings.HasPrefix(u, "//") && !strings.HasPrefix(u, "http://") &&
		!strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "data:")
}
