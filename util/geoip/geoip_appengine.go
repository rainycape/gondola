// +build appengine

package geoip

import (
	"strconv"
	"strings"

	"gnd.la/app"
)

const (
	countryHeaderName     = "X-AppEngine-Country"
	regionHeaderName      = "X-AppEngine-Region"
	cityHeaderName        = "X-AppEngine-City"
	cityLatLongHeaderName = "X-AppEngine-CityLatLong"
)

func country(ctx *app.Context) string {
	return ctx.GetHeader(countryHeaderName)
}

func region(ctx *app.Context) string {
	return ctx.GetHeader(regionHeaderName)
}

func city(ctx *app.Context) string {
	return ctx.GetHeader(cityHeaderName)
}

func latLong(ctx *app.Context) (float64, float64, bool) {
	val := ctx.GetHeader(cityLatLongHeaderName)
	if val != "" {
		p := strings.Split(val, ",")
		if len(p) == 2 {
			lat, err1 := strconv.ParseFloat(p[0], 64)
			lng, err2 := strconv.ParseFloat(p[1], 64)
			if err1 == nil && err2 == nil {
				return lat, lng, true
			}
		}
	}
	return 0, 0, false
}

func loadDatabase(_ *app.App, _ string) error {
	return nil
}
