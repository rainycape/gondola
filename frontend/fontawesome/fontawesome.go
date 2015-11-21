// Package fontawesome defines template assets and functions for
// using fontawesome (see http://fontawesome.io).
//
// Importing this package registers a template asset with the following
// format:
//
//  fontawesome:<version>
//
// Where version is the Font Awesome version you want to use.
//
// Additionally, this package defines the following template function:
//
//  fa <string>: returns the font awesome 4 (and hopefully future versions) icon named by string
//  e.g. {{ fa "external-link" } => <i class="fa fa-external-link"></i>
//
package fontawesome

import (
	"fmt"

	"gnd.la/template/assets"

	"github.com/rainycape/semver"
)

const (
	fontAwesomeFmt = "//netdna.bootstrapcdn.com/font-awesome/%s/css/font-awesome.min.css"
)

func fontAwesomeParser(m *assets.Manager, version string, opts assets.Options) ([]*assets.Asset, error) {
	faVersion, err := semver.Parse(version)
	if err != nil || faVersion.Major != 4 || faVersion.PreRelease != "" || faVersion.Build != "" {
		return nil, fmt.Errorf("invalid font awesome version %q, must in 4.x.y form", faVersion)
	}
	return []*assets.Asset{assets.CSS(fmt.Sprintf(fontAwesomeFmt, version))}, nil
}

func init() {
	assets.Register("fontawesome", assets.SingleParser(fontAwesomeParser))
}
