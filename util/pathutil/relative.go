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
// Note that when running tests (from e.g. go test) or go run
// (e.g. go run myfile.go), this function will return the path relative
// to the current directory rather than the binary. This is done in
// order to allow functions which use relative paths to work under
// those circumstances, since go puts the binaries in a temporary
// directory when using go test or go run, but runs them from the
// current directory.
func Relative(name string) string {
	var full string
	rel := filepath.FromSlash(name)
	if internal.InTest() || internal.InAppEngine() || internal.IsGoRun() {
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
