package util

import (
	"os"
	"path"
)

// RelativePath returns the given path
// relative to the application binary
// e.g.
// binary is at /home/fiam/example/example
// RelativePath("foo") returns /home/fiam/example/foo
// RelativePath("foo/bar") returns /home/fiam/example/foo/bar
// RelativePath("/foo/bar") returns /home/fiam/example/foo/bar
func RelativePath(name string) string {
	wd, _ := os.Getwd()
	var full string
	if path.IsAbs(os.Args[0]) {
		full = path.Join(path.Dir(os.Args[0]), name)
	} else {
		full = path.Join(wd, path.Dir(os.Args[0]), name)
	}
	return path.Clean(full)
}
