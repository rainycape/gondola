package app

import (
	"net/http"
	"testing"
)

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
