package util

import (
	"os"
	"path"
)

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
