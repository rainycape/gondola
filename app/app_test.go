package app

import (
	"fmt"
	"net/http"
	"testing"
)

func helloHandler(ctx *Context) {
	ctx.Write([]byte("Hello world"))
}

func testReverse(t *testing.T, expected string, a *App, name string, args ...interface{}) {
	rev, err := a.Reverse(name, args...)
	if expected != "" {
		if err != nil {
			t.Error(err)
		}
	} else {
		if err == nil {
			t.Errorf("Expecting error while reversing %s with arguments %v", name, args)
		}
	}
	if rev != expected {
		t.Errorf("Error reversing %q with arguments %v, expected %q, got %q", name, args, expected, rev)
	} else {
		t.Logf("Reversed %q with %v to %q", name, args, rev)
	}
}

func TestReverse(t *testing.T) {
	a := New()
	a.HandleOptions("^/program/(\\d+)/$", helloHandler, &Options{Name: "program"})
	a.HandleOptions("^/program/(\\d+)/version/(\\d+)/$", helloHandler, &Options{Name: "programversion"})
	a.HandleOptions("^/program/(?P<pid>\\d+)/version/(?P<vers>\\d+)/$", helloHandler, &Options{Name: "programversionnamed"})
	a.HandleOptions("^/program/(\\d+)/(?:version/(\\d+)/)?$", helloHandler, &Options{Name: "programoptversion"})
	a.HandleOptions("^/program/(\\d+)/(?:version/(\\d+)/)?(?:revision/(\\d+)/)?$", helloHandler, &Options{Name: "programrevision"})
	a.HandleOptions("^/archive/(\\d+)?$", helloHandler, &Options{Name: "archive"})
	a.HandleOptions("^/history/$", helloHandler, &Options{Name: "history"})
	a.HandleOptions("^/image/(\\w+)\\.(\\w+)$", helloHandler, &Options{Name: "image"})
	a.HandleOptions("^/image/(\\w+)\\-(\\w+)$", helloHandler, &Options{Name: "imagedash"})
	a.HandleOptions("^/image/(\\w+)\\\\(\\w+)$", helloHandler, &Options{Name: "imageslash"})

	testReverse(t, "/program/1/", a, "program", 1)
	testReverse(t, "/program/1/version/2/", a, "programversion", 1, 2)
	testReverse(t, "/program/1/version/2/", a, "programversionnamed", 1, 2)
	testReverse(t, "/program/1/", a, "programoptversion", 1)
	testReverse(t, "/program/1/version/2/", a, "programoptversion", 1, 2)
	testReverse(t, "/program/1/", a, "programrevision", 1)
	testReverse(t, "/program/1/version/2/", a, "programrevision", 1, 2)
	testReverse(t, "/program/1/version/2/revision/3/", a, "programrevision", 1, 2, 3)

	testReverse(t, "/archive/19700101", a, "archive", "19700101")
	testReverse(t, "/archive/", a, "archive")
	testReverse(t, "/history/", a, "history")

	// TODO: These don't work
	/*
		m.HandleOptions("^/section/(sub/(\\d+)/subsub(\\d+))?$", helloHandler, "section")
		testReverse(t, "/section/", a, "section")
		testReverse(t, "/section/sub/1/subsub/2", a, "section", 1, 2)
		testReverse(t, "/section/sub/1", a, "section", 1)
	*/

	// Test invalid reverses
	testReverse(t, "", a, "program")
	testReverse(t, "", a, "program", "foo")
	testReverse(t, "", a, "program", 1, 2)
	testReverse(t, "", a, "programrevision", 1, 2, 3, 4)

	// Dot, dash and slash
	testReverse(t, "/image/test.png", a, "image", "test", "png")
	testReverse(t, "/image/test-png", a, "imagedash", "test", "png")
	testReverse(t, "/image/test\\png", a, "imageslash", "test", "png")
}

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
	url := fmt.Sprintf("http://localhost:%d/", DefaultPort())
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
