package doc

import (
	"errors"
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gnd.la/kvs"
)

var (
	filepathSeparator = string(filepath.Separator)
	getEnvironment    func(kvs.Storage) *Environment
	setEnvironment    func(kvs.Storage, *Environment)
)

type Environment struct {
	kvs.KVS
	Context       build.Context
	Separator     string
	reverseDoc    func(string) string
	reverseSource func(string) string
	cache         map[string]interface{}
}

func NewEnvironment(reverseDoc, reverseSource func(string) string) *Environment {
	if reverseDoc == nil {
		panic(errors.New("reverseDoc is nil"))
	}
	if reverseSource == nil {
		panic(errors.New("reverseSource is nil"))
	}
	return &Environment{
		Context:       build.Default,
		Separator:     filepathSeparator,
		reverseDoc:    reverseDoc,
		reverseSource: reverseSource,
	}
}

func (e Environment) Join(elem ...string) string {
	if e.Context.JoinPath != nil {
		return e.Context.JoinPath(elem...)
	}
	for ii, v := range elem {
		elem[ii] = filepath.FromSlash(v)
	}
	return filepath.Join(elem...)
}

func (e Environment) IsAbs(p string) bool {
	if e.Context.IsAbsPath != nil {
		return e.Context.IsAbsPath(p)
	}
	return filepath.IsAbs(p)
}

func (e Environment) IsDir(p string) bool {
	if e.Context.IsDir != nil {
		return e.Context.IsDir(p)
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func (e Environment) ReadDir(p string) ([]os.FileInfo, error) {
	if e.Context.ReadDir != nil {
		return e.Context.ReadDir(p)
	}
	return ioutil.ReadDir(p)
}

func (e Environment) OpenFile(path string) (io.ReadCloser, error) {
	if e.Context.OpenFile != nil {
		return e.Context.OpenFile(path)
	}
	return os.Open(path)
}

func (e Environment) FromSlash(p string) string {
	r := filepath.FromSlash(p)
	if e.Separator != filepathSeparator {
		r = strings.Replace(r, filepathSeparator, e.Separator, -1)
	}
	return r
}

func (e Environment) Dir(p string) string {
	if e.Separator != filepathSeparator {
		p = strings.Replace(p, e.Separator, filepathSeparator, -1)
	}
	return filepath.Dir(p)
}

func (e Environment) Base(p string) string {
	if e.Separator != filepathSeparator {
		p = strings.Replace(p, e.Separator, filepathSeparator, -1)
	}
	return filepath.Base(p)
}

func GetEnvironment(s kvs.Storage) *Environment      { return getEnvironment(s) }
func SetEnvironment(s kvs.Storage, env *Environment) { setEnvironment(s, env) }

func init() {
	kvs.TypeFuncs(&getEnvironment, &setEnvironment)
}
