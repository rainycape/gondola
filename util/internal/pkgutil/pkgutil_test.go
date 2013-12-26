package pkgutil

import (
	"testing"
)

func TestSplit(t *testing.T) {
	cases := map[string][]string{
		"gnd.la/i18n.T":        {"gnd.la/i18n", "T"},
		"gnd.la/app.Context.T": {"gnd.la/app", "Context.T"},
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
