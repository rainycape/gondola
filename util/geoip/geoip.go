// Package geoip provides allows retrieving geographical information
// from an incoming request.
//
// To use this package outside of App Engine, a GeoIP database
// is required. See Load for more information.
//
package geoip

import (
	"gnd.la/app"
)

// Country returns the country for the IP address associated
// with the given *app.Context, as an ISO
func Country(ctx *app.Context) string {
	return country(ctx)
}

func Region(ctx *app.Context) string {
	return region(ctx)
}

// City returns the city name for the IP address associated
// with the given *app.Context.
func City(ctx *app.Context) string {
	return city(ctx)
}

// LatLong returns the latitude and longitude for the IP
// address associated with the given *app.Context. The last
// return value indicates if the coordinates are known.
func LatLong(ctx *app.Context) (float64, float64, bool) {
	return latLong(ctx)
}

// Load loads the given GeoIP2 database from the given filename
// into the given *app.App. Databases can be downloaded from free
// from http://dev.maxmind.com/geoip/geoip2/geolite2/.
//
// Note that the filename might point to either a bare .mmdb file
// or a gzip-compressed .mmdb.gz file.
//
// On some platforms, notably App Engine, the filename argument might
// be empty, because the platform already provides GeoIP functionality.
func Load(a *app.App, filename string) error {
	return loadDatabase(a, filename)
}

// MustLoad works like Load, but panics if there's an error.
func MustLoad(a *app.App, filename string) {
	if err := Load(a, filename); err != nil {
		panic(err)
	}
}
