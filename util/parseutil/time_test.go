package parseutil

import (
	"testing"
	"time"
)

var (
	timeTests = map[string]time.Time{
		"2006-01-02":                time.Date(2006, 01, 02, 0, 0, 0, 0, time.Local),
		"2006-01-02+07:00":          time.Date(2006, 01, 02, 0, 0, 0, 0, time.FixedZone("test", 7*3600)),
		"2006-01-02T15:04:05":       time.Date(2006, 01, 02, 15, 4, 5, 0, time.Local),
		"2006-01-02T15:04:05+07:00": time.Date(2006, 01, 02, 15, 4, 5, 0, time.FixedZone("test", 7*3600)),
		//  - 2006-01-02T15:04:05.999999999Z07:00 (time.RFC3339Nano)
	}
)

func TestParseTime(t *testing.T) {
	for k, v := range timeTests {
		val, err := DateTime(k)
		if err != nil {
			t.Error(err)
			continue
		}
		if !val.Equal(v) {
			t.Errorf("expecting %s when parsing %q, got %s instead", v, k, val)
		}
	}
}
