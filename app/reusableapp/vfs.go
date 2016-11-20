package reusableapp

import (
	"errors"
	"os"
	"strings"

	"github.com/rainycape/vfs"

	"gnd.la/util/stringutil"
	"gnd.la/util/vfsutil"
)

func openVFS(appName string, abspath string, baked string) (vfsutil.VFS, error) {
	if appName == "" {
		return nil, errors.New("empty app name")
	}
	if len(baked) > 0 {
		noVFS := false
		fields, _ := stringutil.SplitFields(os.Getenv("GONDOLA_APP_NO_BAKED_DATA"), " ")
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
	return vfs.FS(abspath)
}
