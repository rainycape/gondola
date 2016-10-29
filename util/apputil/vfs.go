// Package apputil contains functions for writing reusable apps.
package apputil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rainycape/vfs"

	"gnd.la/util/stringutil"
	"gnd.la/util/vfsutil"
)

// OpenVFS opens a VFS for a reusable app. Since reusable apps need to have
// their non-go files baked, they should use this function for opening their
// assets and templates. This function will initialize the VFS with the baked
// resources when the following conditions are met:
//
// - baked is non-empty
// - the environment variable GONDOLA_APP_NO_VFS doesn't contain the app name (appName argument)
//
// Otherwise, a VFS pointing the the directory indicated by relativePath (relative to
// the source file calling this funciton) will be returned. This is useful while writing
// reusable apps, so you can start the development server with:
//
//	GONDOLA_APP_NO_VFS=MyApp
//
// And then OpenVFS will use the assets from the filesystem. Once you're ready to
// commit the changes, you can regenerate the baked assets via gondola bake
// (usually called from go generate) and users of your app will be able to just
// go get it.
func OpenVFS(appName string, relativePath string, baked string) (vfsutil.VFS, error) {
	if appName == "" {
		return nil, errors.New("empty app name")
	}
	if len(baked) > 0 {
		noVFS := false
		fields, _ := stringutil.SplitFields(os.Getenv("GONDOLA_APP_NO_VFS"), " ")
		lowerAppName := strings.ToLower(appName)
		for _, field := range fields {
			if strings.ToLower(field) == lowerAppName {
				noVFS = true
			}
		}
		if !noVFS {
			return vfsutil.OpenBaked(baked)
		}
	}
	_, file, _, ok := runtime.Caller(2)
	if !ok {
		return nil, fmt.Errorf("could not determine relative path for assets from app %s", appName)
	}
	dir := filepath.Dir(file)
	root := filepath.Join(dir, filepath.FromSlash(relativePath))
	return vfs.FS(root)
}

// MustOpenVFS is a shorthand for OpenAppVFS which panics in case of error.
func MustOpenVFS(appName string, relativePath string, baked string) vfsutil.VFS {
	fs, err := OpenVFS(appName, relativePath, baked)
	if err != nil {
		panic(err)
	}
	return fs
}
