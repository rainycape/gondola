package assets

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"gnd.la/log"
	"gnd.la/util/formatutil"
)

var (
	errNoAssets = errors.New("no assets to bundle")
	urlRe       = regexp.MustCompile("i?url\\s*?\\((.*?)\\)")
)

func bundleName(groups []*Group, ext string, o Options) (string, error) {
	h := fnv.New32a()
	for _, group := range groups {
		for _, asset := range group.Assets {
			code, err := asset.Code(group.Manager)
			if err != nil {
				return "", err
			}
			io.WriteString(h, code)
		}
	}
	io.WriteString(h, o.String())
	sum := hex.EncodeToString(h.Sum(nil))
	name := groups[0].Assets[0].Name
	if ext == "" {
		ext = path.Ext(name)
	} else {
		ext = "." + ext
	}
	return path.Join(path.Dir(name), "bundle.gen."+sum+ext), nil
}

func Bundle(groups []*Group, opts Options) (*Asset, error) {
	assetType := Type(-1)
	var names []string
	for _, group := range groups {
		for _, v := range group.Assets {
			if v.Type == TypeOther {
				return nil, fmt.Errorf("asset %q does not specify a Type and can't be bundled", v.Name)
			}
			if assetType < 0 {
				assetType = v.Type
			} else if assetType != v.Type {
				return nil, fmt.Errorf("asset %q has different code type %s (first asset is of type %s)", v.Name, v.Type, assetType)
			}
			names = append(names, v.Name)
		}
	}
	bundler := bundlers[assetType]
	if bundler == nil {
		return nil, fmt.Errorf("no bundler for %s", assetType)
	}
	// Prepare the code, changing relative paths if required
	name, err := bundleName(groups, assetType.Ext(), opts)
	if err != nil {
		return nil, err
	}
	// The bundle is output to the first manager
	m := groups[0].Manager
	// Check if the code has been already bundled
	if f, _, _ := m.Load(name); f != nil {
		f.Close()
		log.Debugf("%s already bundled into %s and up to date", names, name)
	} else {
		dir := path.Dir(name)
		log.Debugf("bundling %v", names)
		var code []string
		for _, group := range groups {
			for _, v := range group.Assets {
				c, err := v.Code(group.Manager)
				if err != nil {
					return nil, fmt.Errorf("error getting code for asset %q: %s", v.Name, err)
				}
				if vd := path.Dir(v.Name); vd != dir {
					if assetType == TypeCSS {
						log.Debugf("asset %q will move from %v to %v, rewriting relative paths...", v.Name, vd, dir)
						c = replaceRelativePaths(c, vd, dir)
					} else {
						log.Warningf("asset %q will move from %v to %v, relative paths might not work", v.Name, vd, dir)
					}
				}
				code = append(code, c)
			}
		}
		// Bundle to a buf first. We don't want to create
		// the file if the bundling fails.
		var buf bytes.Buffer
		allCode := strings.Join(code, "\n\n")
		reader := strings.NewReader(allCode)
		if err := bundler.Bundle(&buf, reader, opts); err != nil {
			return nil, err
		}
		s := makeLinksCacheable(m, dir, buf.Bytes())
		initial := len(allCode)
		final := len(s)
		var percent float64
		if initial != 0 {
			percent = float64(final) / float64(initial) * 100
		}
		log.Debugf("reduced size from %s to %s (%.2f%%)", formatutil.Size(uint64(initial)),
			formatutil.Size(uint64(final)), percent)
		w, err := m.Create(name, true)
		if err == nil {
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
	return &Asset{
		Name:     name,
		Type:     assetType,
		Position: groups[0].Assets[0].Position,
	}, nil
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
