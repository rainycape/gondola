package stringutil

import (
	"testing"
)

const (
	// these make the probability of collissions
	// small enough, since we're dealing with
	// 26^100 and 256^100 combinations respectivelly
	rIterations = 100000
	rN          = 100
)

func TestRandomString(t *testing.T) {
	generated := make(map[string]bool)
	for ii := 0; ii < rIterations; ii++ {
		rand := Random(rN)
		if len(rand) != rN {
			t.Errorf("expecting %d characters, got %d", rN, len(rand))
		}
		if generated[rand] {
			t.Errorf("duplicated value %s", rand)
		}
		generated[rand] = true
	}
}

func TestRandomBytes(t *testing.T) {
	generated := make(map[string]bool)
	for ii := 0; ii < rIterations; ii++ {
		rand := RandomBytes(rN)
		if len(rand) != rN {
			t.Errorf("expecting %d characters, got %d", rN, len(rand))
		}
		if generated[string(rand)] {
			t.Errorf("duplicated value %s", rand)
		}
		generated[string(rand)] = true
	}
}
