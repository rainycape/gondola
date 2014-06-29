package app

import (
	"fmt"
	"net/http"
	"testing"
)

func testApp(nolog bool) (*App, string) {
	app := New()
	if nolog {
		app.Logger = nil
	}
	f := func(ctx *Context) {}
	app.Handle("^/foobar/$", f)
	app.Handle("^/foobar2/$", f)
	app.Handle("^/foobar3/$", f)
	app.Handle("^/foobar4/$", f)
	app.Handle("^/foobar5/$", f)
	app.Handle("^/article/(\\d)$", f)
	app.Handle("^/$", f)
	url := fmt.Sprintf("http://localhost:%d/", app.Config().Port)
	return app, url
}

func benchmarkServe(b *testing.B, nolog bool) {
	a, url := testApp(nolog)
	go func() {
		a.ListenAndServe()
	}()
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		_, err := http.Get(url)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServe(b *testing.B) {
	benchmarkServe(b, false)
}

func BenchmarkServeNoLog(b *testing.B) {
	benchmarkServe(b, true)
}

func benchmarkDirect(b *testing.B, path string, nolog bool) {
	app, url := testApp(nolog)
	if path != "" {
		url += path
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		app.ServeHTTP(nil, req)
	}
}

func BenchmarkDirect(b *testing.B) {
	benchmarkDirect(b, "", false)
}

func BenchmarkDirectNoLog(b *testing.B) {
	benchmarkDirect(b, "", true)
}

func BenchmarkDirectReNoLog(b *testing.B) {
	benchmarkDirect(b, "article/7", true)
}
