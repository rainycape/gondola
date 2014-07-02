package doc

import (
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gnd.la/app"
)

var (
	DefaultContext    = Context{Context: build.Default, Separator: string(filepath.Separator)}
	filepathSeparator = string(filepath.Separator)
)

type Context struct {
	build.Context
	Separator         string
	App               *app.App
	SourceHandlerName string
	DocHandlerName    string
	cache             map[string]interface{}
}

func (c Context) Join(elem ...string) string {
	if c.Context.JoinPath != nil {
		return c.Context.JoinPath(elem...)
	}
	for ii, v := range elem {
		elem[ii] = filepath.FromSlash(v)
	}
	return filepath.Join(elem...)
}

func (c Context) IsAbs(p string) bool {
	if c.Context.IsAbsPath != nil {
		return c.Context.IsAbsPath(p)
	}
	return filepath.IsAbs(p)
}

func (c Context) IsDir(p string) bool {
	if c.Context.IsDir != nil {
		return c.Context.IsDir(p)
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func (c Context) ReadDir(p string) ([]os.FileInfo, error) {
	if c.Context.ReadDir != nil {
		return c.Context.ReadDir(p)
	}
	return ioutil.ReadDir(p)
}

func (c Context) OpenFile(path string) (io.ReadCloser, error) {
	if c.Context.OpenFile != nil {
		return c.Context.OpenFile(path)
	}
	return os.Open(path)
}

func (c Context) FromSlash(p string) string {
	r := filepath.FromSlash(p)
	if c.Separator != filepathSeparator {
		r = strings.Replace(r, filepathSeparator, c.Separator, -1)
	}
	return r
}

func (c Context) Dir(p string) string {
	if c.Separator != filepathSeparator {
		p = strings.Replace(p, c.Separator, filepathSeparator, -1)
	}
	return filepath.Dir(p)
}

func (c Context) Base(p string) string {
	if c.Separator != filepathSeparator {
		p = strings.Replace(p, c.Separator, filepathSeparator, -1)
	}
	return filepath.Base(p)
}
