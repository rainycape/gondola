package app_test

import (
	"fmt"
	"gnd.la/app"
	"gnd.la/app/tester"
	"testing"
)

func TestAppendSlash(t *testing.T) {
	a := app.New()
	a.Handle("/foo/", func(ctx *app.Context) {
		ctx.WriteString("Hello world")
	})
	tt := tester.New(t, a)
	tt.Get("/foo", nil).Expect(301).ExpectHeader("Location", "/foo/")
	a.SetAppendSlash(false)
	tt.Get("/foo", nil).Expect(404)
}

func TestXHeaders(t *testing.T) {
	a := app.New()
	a.Handle("/", func(ctx *app.Context) {
		fmt.Fprintf(ctx, "%s\n%s", ctx.RemoteAddress(), ctx.URL().String())
	})
	tt := tester.New(t, a)
	tt.Get("/", nil).AddHeader("X-Real-IP", "8.8.8.8").AddHeader("X-Scheme", "https").Expect("\nhttp://localhost/")
	a.SetTrustXHeaders(true)
	tt.Get("/", nil).AddHeader("X-Real-IP", "8.8.8.8").AddHeader("X-Scheme", "https").Expect("8.8.8.8\nhttps://localhost/")
}
