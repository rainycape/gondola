// Package vfsutil implements utility functions for working
// with virtual filesystems.
package vfsutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/rainycape/vfs"
	"gnd.la/log"
)

func MemFromDir(dir string) vfs.VFS {
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

func OpenBaked(s string) vfs.VFS {
	fs, err := vfs.TarGzip(strings.NewReader(s))
	if err != nil {
		panic(err)
	}
	return fs
}

func BakedFS(w io.Writer, dir string, extensions []string) error {
	var buf bytes.Buffer
	if err := Bake(&buf, dir, extensions); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "vfsutil.OpenBaked(%q)\n", buf.String())
	return err

}

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
