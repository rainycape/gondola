package util

import (
	"gnd.la/util/internal"
	"os"
	"path/filepath"
)

// RelativePath returns the given path
// relative to the application binary
// e.g.
// binary is at /home/fiam/example/example
// RelativePath("foo") returns /home/fiam/example/foo
// RelativePath("foo/bar") returns /home/fiam/example/foo/bar
// RelativePath("/foo/bar") returns /home/fiam/example/foo/bar.
// Note that when running tests (from e.g. go test), this function
// will return the path relative to the current directory rather
// than the binary. This is done in order to allow functions which
// use relative paths to work while being tested, since go test puts
// the test binary in a temporary directory, but runs it in the
// package directory.
func RelativePath(name string) string {
	var full string
	rel := filepath.FromSlash(name)
	wd, _ := os.Getwd()
	if internal.InTest() {
		full = filepath.Join(wd, rel)
	} else {
		if filepath.IsAbs(os.Args[0]) {
			full = filepath.Join(filepath.Dir(os.Args[0]), rel)
		} else {
			full = filepath.Join(wd, filepath.Dir(os.Args[0]), rel)
		}
	}
	return filepath.Clean(full)
}
