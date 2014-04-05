package parseutil

import (
	"strings"
	"time"
)

// DateTime parses a date with, optionally a time
// and a timezone in an RFC3339 style. The accepted
// formats are:
//
//  - 2006-01-02 (interpred as 0 hour, 0 minute and 0 second)
//  - 2006-01-02Z07:00 (same as the previous one, but with TZ)
//  - 2006-01-02T15:04:05
//  - 2006-01-02T15:04:05Z07:00 (time.RFC3339)
//  - 2006-01-02T15:04:05.999999999Z07:00 (time.RFC3339Nano)
//
// In all dates, the second number indicates the month and the third,
// the day. Number values less than ten must have a leading
// zero.
//
// Note that the format Z07:00 indicates that the timezone can be
// specified as either Z (UTC) or either (+|-)hh:mm.
func DateTime(value string) (time.Time, error) {
	if strings.Contains(value, "T") {
		if containsTz(value) {
			if strings.Contains(value, ".") {
				return time.Parse(time.RFC3339Nano, value)
			}
			return time.Parse(time.RFC3339, value)
		}
		return time.Parse("2006-01-02T15:04:05", value)
	}
	if containsTz(value) {
		return time.Parse("2006-01-02Z07:00", value)
	}
	return time.Parse("2006-01-02", value)
}

func containsTz(value string) bool {
	if strings.Contains(value, "Z") {
		return true
	}
	// This removes the date from the start, so any + or -
	// character indicates TZ.
	rem := value[10:]
	return strings.Contains(rem, "-") || strings.Contains(rem, "+")
}
