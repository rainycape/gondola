// Package httpserve contains constants and utility functions for serving
// HTTP content.
package httpserve

import (
	"net/http"
	"strconv"
	"time"
)

const (
	// MaxCacheControlAge is the maximum age in a cache-control
	// header supported by most browsers. Notably, IE9 will consider
	// as stale resources with a max-age greather than this value
	// (2^31).
	MaxCacheControlAge = int(^uint32(0) >> 1)
)

var (
	// MaxExpires is the maximum safe HTTP Expires header value.
	// It represents the UNIX biggest timestamp representable with
	// 32 bits.
	MaxExpires = time.Unix(int64(MaxCacheControlAge), 0).UTC()

	maxExpiresValue         = MaxExpires.Format(time.RFC1123)
	maxCacheControlAgeValue = strconv.Itoa(MaxCacheControlAge)
)

// NeverExpires sets the appropriate headers on the given http.ResponseWriter
// to make the response never expire (in practical terms, it really expires in
// 68 years).
func NeverExpires(w http.ResponseWriter) {
	header := w.Header()
	header.Add("Cache-Control", "max-age="+maxCacheControlAgeValue)
	header.Add("Expires", maxExpiresValue)
}
