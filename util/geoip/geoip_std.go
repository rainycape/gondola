// +build !appengine

package geoip

import (
	"gnd.la/app"
	"gnd.la/log"

	"github.com/rainycape/geoip"
)

const (
	geoIPKey  = "__gondola_geoip"
	recordKey = "__gondola_record"
)

func getGeoIPRecord(ctx *app.Context) *geoip.Record {
	if rec, _ := ctx.Get(recordKey).(*geoip.Record); rec != nil {
		return rec
	}
	g, _ := ctx.App().Get(geoIPKey).(*geoip.GeoIP)
	if g == nil {
		log.Warning("no GeoIP data loaded - did you call geoip.Load()?")
		return nil
	}
	rec, _ := g.Lookup(ctx.RemoteAddress())
	ctx.Set(recordKey, rec)
	return rec
}

func country(ctx *app.Context) string {
	rec := getGeoIPRecord(ctx)
	if rec != nil && rec.Country != nil {
		return rec.Country.Code
	}
	return ""
}

func region(ctx *app.Context) string {
	rec := getGeoIPRecord(ctx)
	if rec != nil && len(rec.Subdivisions) > 0 {
		return rec.Subdivisions[0].Name.String()
	}
	return ""
}

func city(ctx *app.Context) string {
	rec := getGeoIPRecord(ctx)
	if rec != nil && rec.City != nil {
		return rec.City.Name.String()
	}
	return ""
}

func latLong(ctx *app.Context) (float64, float64, bool) {
	rec := getGeoIPRecord(ctx)
	if rec != nil && (rec.Latitude != 0 || rec.Longitude != 0) {
		return rec.Latitude, rec.Longitude, true
	}
	return 0, 0, false
}

func loadDatabase(a *app.App, filename string) error {
	g, err := geoip.Open(filename)
	if err != nil {
		return err
	}
	a.Set(geoIPKey, g)
	return nil
}
