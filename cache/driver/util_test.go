package driver

import (
	"testing"
)

type pe struct {
	Port     int
	Expected string
}

func TestDefaultPort(t *testing.T) {
	cases := map[string]pe{
		"10.12.45.37":                   {11211, "10.12.45.37:11211"},
		"10.12.45.37:11212":             {11211, "10.12.45.37:11212"},
		"1fff:0:a88:85a3::ac1f":         {11211, "[1fff:0:a88:85a3::ac1f]:11211"},
		"[1fff:0:a88:85a3::ac1f]":       {11211, "[1fff:0:a88:85a3::ac1f]:11211"},
		"[1fff:0:a88:85a3::ac1f":        {11211, "[1fff:0:a88:85a3::ac1f]:11211"},
		"[1fff:0:a88:85a3::ac1f]:11212": {11211, "[1fff:0:a88:85a3::ac1f]:11212"},
		"www.google.com":                {80, "www.google.com:80"},
		"www.google.com:81":             {80, "www.google.com:81"},
	}
	for k, v := range cases {
		val := DefaultPort(k, v.Port)
		if val != v.Expected {
			t.Errorf("Error adding port %d to %q. Want %q, got %q.", v.Port, k, v.Expected, val)
			continue
		}
		t.Logf("%s with port %d => %s", k, v.Port, val)
	}
}
