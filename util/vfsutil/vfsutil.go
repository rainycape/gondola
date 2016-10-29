// Package vfsutil contains small utility functions for working with
// virtual filesystems. See the docs for github.com/rainycape/vfs
// for more information.
package vfsutil

import (
	"io"
	"os"
	"path"
	"strings"

	"github.com/rainycape/vfs"
	"gnd.la/log"
)

// VFS is just an alias for github.com/rainycape/vfs.VFS, to
// avoid importing the latter.
type VFS interface {
	vfs.VFS
}

func MemFromDir(dir string) VFS {
	fs, err := vfs.FS(dir)
	if err != nil {
		panic(err)
	}
	mem := vfs.Memory()
	if err := vfs.Clone(mem, fs); err != nil {
		panic(err)
	}
	return mem
}

// OpenBaked opens a VFS baked into a string using gondola bake
func OpenBaked(s string) (VFS, error) {
	return vfs.TarGzip(strings.NewReader(s))
}

// MustOpenBaked is a shorthand for OpenBaked() which panics.
func MustOpenBaked(s string) VFS {
	fs, err := OpenBaked(s)
	if err != nil {
		panic(err)
	}
	return fs
}

// Bake writes the data for a VFS generated from dir to the given
// io.Writer. The extensions argument can be used to limit the
// files included in the VFS by their extension. If empty, all
// files are included.
func Bake(w io.Writer, dir string, extensions []string) error {
	fs, err := vfs.FS(dir)
	if err != nil {
		return err
	}
	if len(extensions) > 0 {
		// Clone the fs and remove files not matching the extension
		exts := make(map[string]bool)
		for _, v := range extensions {
			if v == "" {
				continue
			}
			if v[0] != '.' {
				v = "." + v
			}
			exts[strings.ToLower(v)] = true
		}
		mem := vfs.Memory()
		if err := vfs.Clone(mem, fs); err != nil {
			return err
		}
		err := vfs.Walk(mem, "/", func(fs vfs.VFS, p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			if !exts[strings.ToLower(path.Ext(p))] {
				if err := fs.Remove(p); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		fs = mem
	}
	vfs.Walk(fs, "/", func(_ vfs.VFS, p string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			log.Debugf("baking %s", p)
		}
		return nil
	})
	return vfs.WriteTarGzip(w, fs)
}
