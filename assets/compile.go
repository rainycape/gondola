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

const (
	// If you define your own code types, use numbers > 1000
	CodeTypeCss        = 1
	CodeTypeJavascript = 2
)

var (
	ErrNoAssets = errors.New("no assets to compile")
	urlRe       = regexp.MustCompile("i?url\\s*?\\((.*?)\\)")
)

type CodeAsset interface {
	Asset
	CodeType() int
	Code() (string, error)
}

type CodeAssetList []CodeAsset

func (c CodeAssetList) Names() []string {
	var names []string
	for _, v := range c {
		names = append(names, v.Name())
	}
	return names
}

func (c CodeAssetList) CompiledName(ext string, o Options) string {
	if len(c) == 0 {
		return ""
	}
	h := fnv.New32a()
	for _, v := range c {
		code, err := v.Code()
		if err != nil {
			// TODO: Do something better here
			panic(err)
		}
		io.WriteString(h, code)
	}
	io.WriteString(h, o.String())
	sum := hex.EncodeToString(h.Sum(nil))
	name := c[0].Name()
	if ext == "" {
		ext = path.Ext(name)
	} else {
		ext = "." + ext
	}
	return path.Join(path.Dir(name), "asset-"+sum+ext)
}

func Compile(m Manager, assets []Asset, opts Options) ([]Asset, error) {
	if len(assets) == 0 {
		return nil, ErrNoAssets
	}
	var ctype int
	codeAssets := make(CodeAssetList, len(assets))
	for ii, v := range assets {
		c, ok := v.(CodeAsset)
		if !ok {
			return nil, fmt.Errorf("asset %q (type %T) does not implement CodeAsset and can't be compiled", v.Name(), v)
		}
		if ctype == 0 {
			ctype = c.CodeType()
		} else if ctype != c.CodeType() {
			return nil, fmt.Errorf("asset %q has different code type %d (first asset is of type %d)", v.Name(), c.CodeType(), ctype)
		}
		codeAssets[ii] = c
	}
	compiler := compilers[ctype]
	if compiler == nil {
		return nil, fmt.Errorf("no compiler for code type %d", ctype)
	}
	// Prepare the code, changing relative paths if required
	name := codeAssets.CompiledName(compiler.Ext(), opts)
	dir := path.Dir(name)
	var code []string
	for _, v := range codeAssets {
		c, err := v.Code()
		if err != nil {
			return nil, fmt.Errorf("error getting code for asset %q: %s", v.Name(), err)
		}
		if vd := path.Dir(v.Name()); vd != dir {
			if ctype == CodeTypeCss {
				log.Debugf("asset %q will move from %v to %v, rewriting relative paths...", v.Name(), vd, dir)
				c = replaceRelativePaths(c, vd, dir)
			} else {
				log.Warningf("asset %q will move from %v to %v, relative paths might not work", v.Name(), vd, dir)
			}
		}
		code = append(code, c)
	}
	// Check if the code has been already compiled
	if _, _, err := m.Load(name); err == nil {
		log.Debugf("%s already compiled into %s and up to date", codeAssets.Names(), name)
	} else {
		log.Debugf("Compiling %v", codeAssets.Names())
		// Compile to a buf first. We don't want to create
		// the file if the compilation fails
		var buf bytes.Buffer
		reader := strings.NewReader(strings.Join(code, "\n\n"))
		if err := compiler.Compile(reader, &buf, m, opts); err != nil {
			return nil, err
		}
		w, err := m.Create(name)
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
	asset, err := compiler.Asset(name, m, opts)
	if err != nil {
		return nil, err
	}
	return []Asset{asset}, nil
}

func makeLinksCacheable(m Manager, dir string, b []byte) string {
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
