package pathutil

import (
	"gnd.la/internal"
	"os"
	"path/filepath"
)

var (
	// initial working directory
	iwd string
)

// Relative returns the given path
// relative to the application binary
// e.g.
// binary is at /home/fiam/example/example
// Relative("foo") returns /home/fiam/example/foo
// Relative("foo/bar") returns /home/fiam/example/foo/bar
// Relative("/foo/bar") returns /home/fiam/example/foo/bar.
// Note that when running tests (from e.g. go test), this function
// will return the path relative to the current directory rather
// than the binary. This is done in order to allow functions which
// use relative paths to work while being tested, since go test puts
// the test binary in a temporary directory, but runs it in the
// package directory.
func Relative(name string) string {
	var full string
	rel := filepath.FromSlash(name)
	if internal.InTest() || internal.InAppEngine() {
		cwd, _ := os.Getwd()
		full = filepath.Join(cwd, rel)
	} else {
		if filepath.IsAbs(os.Args[0]) {
			full = filepath.Join(filepath.Dir(os.Args[0]), rel)
		} else {
			full = filepath.Join(iwd, filepath.Dir(os.Args[0]), rel)
		}
	}
	return filepath.Clean(full)
}

func init() {
	iwd, _ = os.Getwd()
}
