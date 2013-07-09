package pkg

import (
	"testing"
)

func TestSplit(t *testing.T) {
	cases := map[string][]string{
		"gondola/i18n.T":        {"gondola/i18n", "T"},
		"gondola/mux.Context.T": {"gondola/mux", "Context.T"},
	}
	for k, v := range cases {
		pkg, name := SplitQualifiedName(k)
		if pkg != v[0] || name != v[1] {
			t.Errorf("error testing qname %q, expected pkg %q and name %q, got pkg %q and name %q",
				k, v[0], v[1], pkg, name)
		}
	}
}

func TestIsPackage(t *testing.T) {
	cases := map[string]bool{
		".": true,
		"/this-path-hopefully-does-no-exist": false,
	}
	for k, v := range cases {
		is := IsPackage(k)
		if is != v {
			t.Errorf("IsPackage() returned %v for directory %q, want %v", is, k, v)
		}
	}
}
