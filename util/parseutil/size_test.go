package parseutil

import (
	"testing"
)

func TestSize(t *testing.T) {
	sizes := map[string]uint64{
		"0":     0,
		"0GB":   0,
		"0.5K":  512,
		"1.5MB": uint64(1024 * 1024 * 1.5),
	}
	for k, v := range sizes {
		val, err := Size(k)
		if err != nil {
			t.Error(err)
			continue
		}
		if val != v {
			t.Errorf("error parsing %q - want %d, got %d", k, v, val)
		}
	}
}
