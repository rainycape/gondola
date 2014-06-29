package app

import (
	"strconv"
	"testing"
)

func helloHandler(ctx *Context) {
	ctx.Write([]byte("Hello world"))
}

func testReverse(t testing.TB, expected string, a *App, name string, args []interface{}) {
	rev, err := a.Reverse(name, args...)
	if expected != "" {
		if err != nil {
			t.Error(err)
		}
	} else {
		if err == nil {
			t.Errorf("expecting error while reversing %s with arguments %v", name, args)
		}
	}
	if rev != expected {
		t.Errorf("error reversing %q with arguments %v, expected %q, got %q", name, args, expected, rev)
	}
}

func setupReverseTest() (*App, map[string]string) {
	a := New()
	m := make(map[string]string)
	for ii, v := range regexpTests {
		if _, ok := m[v.pattern]; !ok {
			name := strconv.Itoa(ii)
			a.HandleOptions(v.pattern, helloHandler, &Options{Name: name})
			m[v.pattern] = name
		}
	}
	return a, m
}

func runReverseTests(t testing.TB, a *App, m map[string]string) {
	for _, v := range regexpTests {
		name := m[v.pattern]
		testReverse(t, v.expect, a, name, v.args)
	}
}

func TestReverse(t *testing.T) {
	a, m := setupReverseTest()
	runReverseTests(t, a, m)
}

func BenchmarkReverse(b *testing.B) {
	a, m := setupReverseTest()
	var tb testing.TB = b
	b.ReportAllocs()
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		runReverseTests(tb, a, m)
	}
}
