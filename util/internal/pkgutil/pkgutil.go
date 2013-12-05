// Package pkgutil contains some small utilities for working
// with go packages.
package pkgutil

import (
	"os"
	"path/filepath"
	"strings"
)

// SplitQualifiedName splits a qualified name into the package
// and the local name inside the package.
func SplitQualifiedName(qname string) (pkg string, name string) {
	pos := strings.LastIndex(qname, "/")
	if pos == -1 {
		pos = 0
	}
	dot := strings.Index(qname[pos:], ".")
	if dot == -1 {
		return "", ""
	}
	dot += pos
	return qname[:dot], qname[dot+1:]
}

// IsPackage returns wheter the given directory is a Go package.
func IsPackage(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		return false
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		return false
	}
	for _, v := range names {
		if strings.ToLower(filepath.Ext(v)) == ".go" {
			return true
		}
	}
	return false
}

// ListPackages returns a list of packages found under the given
// directory (including itself, if it's a package).
func ListPackages(dir string) ([]string, error) {
	var pkgs []string
	e := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info != nil && info.IsDir() && IsPackage(path) {
			pkgs = append(pkgs, path)
		}
		return nil
	})
	return pkgs, e
}
