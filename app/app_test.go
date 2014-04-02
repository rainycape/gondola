package app_test

import (
	"fmt"
	"gnd.la/app"
	"gnd.la/app/tester"
	"testing"
	"time"
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

func TestGoWait(t *testing.T) {
	a := app.New()
	a.Handle("/(no)?wait", func(ctx *app.Context) {
		value := 42
		ctx.Go(func(bg *app.Context) {
			time.Sleep(time.Second)
			value++
			panic("handled")
		})
		if ctx.IndexValue(0) != "no" {
			ctx.Wait()
		}
		fmt.Fprintf(ctx, "%d", value)
	})
	tt := tester.New(t, a)
	tt.Get("/wait", nil).Expect("43")
	tt.Get("/nowait", nil).Expect("42")
}
