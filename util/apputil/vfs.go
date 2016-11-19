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

func openVFS(appName string, relativePath string, baked string) (vfsutil.VFS, error) {
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
