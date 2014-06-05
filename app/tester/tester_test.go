package tester_test

import (
	"bytes"
	"fmt"
	"gnd.la/app"
	"gnd.la/app/tester"
	"gnd.la/util/generic"
	"gnd.la/util/stringutil"
	"io/ioutil"
	"net/url"
	"sort"
	"strings"
	"testing"
)

var (
	testApp *app.App
)

type reporter struct {
	*testing.T
	err   error
	fatal error
}

func (r *reporter) Error(args ...interface{}) {
	r.err = args[0].(error)
}

func (r *reporter) Fatal(args ...interface{}) {
	r.fatal = args[0].(error)
}

func TestExpect(t *testing.T) {
	tt := tester.New(t, testApp)
	tt.Get("/hello", nil).Expect(200).Contains("hello").Expect("hello world").Match("\\w+ \\w+").
		ExpectHeader("X-Hello", "World").ExpectHeader("X-Number", 42).ContainsHeader("X-Hello", "Wo").
		MatchHeader("X-Hello", "W.*d")
	tt.Post("/does-not-exist", nil).Expect(404)
	echoData := []byte{1, 2, 3, 4, 5, 6}
	tt.Post("/echo", echoData).Expect(echoData)
	tt.Post("/echo", echoData).Expect(bytes.NewReader(echoData))
	tt.Post("/echo", string(echoData)).Expect(echoData)
	tt.Post("/echo", echoData).Expect(string(echoData))
	tt.Post("/echo", bytes.NewReader(echoData)).Expect(echoData)
	tt.Post("/echo", nil).Expect(200).Expect("")
	tt.Post("/echo", nil).Expect(200).Expect(nil)
	form := map[string]interface{}{"foo": 1, "bar": "baz"}
	formExpect := "bar=baz\nfoo=1\n"
	tt.Form("/echo-form", form).Expect(formExpect)
	tt.Get("/echo-form", form).Expect(formExpect)
}

func TestInvalidRegexp(t *testing.T) {
	r := &reporter{T: t}
	tt := tester.New(r, testApp)
	tt.Get("/hello", nil).Match("\\Ga+")
	if r.fatal == nil || !strings.Contains(r.fatal.Error(), "error compiling regular expression") {
		t.Errorf("expecting invalid re error, got %s", r.fatal)
	}
}

func TestInvalidWriteHeader(t *testing.T) {
	r := &reporter{T: t}
	tt := tester.New(r, testApp)
	tt.Get("/invalid-write-header", nil).Expect(nil)
	if r.err == nil || !strings.Contains(r.err.Error(), "WriteHeader() called with invalid code") {
		t.Errorf("expecting invalid WriteHeader() error, got %s", r.err)
	}
}

func TestMultipleWriteHeader(t *testing.T) {
	r := &reporter{T: t}
	tt := tester.New(r, testApp)
	err := tt.Get("/multiple-write-header", nil).Expect(nil).Err()
	if err != r.err {
		t.Errorf("bad error from Err()")
	}
	if r.err == nil || !strings.Contains(r.err.Error(), "WriteHeader() called 2 times") {
		t.Errorf("expecting multiple WriteHeader() error, got %s", r.err)
	}
}

func TestExpectErrors(t *testing.T) {
	r := &reporter{T: t}
	tt := tester.New(r, testApp)
	tt.Get("/hello", nil).Expect(400)
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Get("/hello", nil).Contains("nothing")
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Get("/hello", nil).Expect("nothing")
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Get("/hello", nil).ExpectHeader("X-Hello", 13)
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Get("/hello", nil).ExpectHeader("X-Number", 37)
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Get("/hello", nil).Expect(nil)
	if r.err == nil {
		t.Error("expecting an error")
	}
	something := []byte{1, 2, 3, 4, 5, 6}
	tt.Post("/echo", nil).Expect(something)
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Post("/echo", nil).Expect(bytes.NewReader(something))
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Post("/echo", something).Expect(nil)
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Post("/echo", something).Expect(float64(0))
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Post("/echo", float64(0)).Expect(float64(0))
	if r.fatal == nil {
		t.Error("expecting a fatal error")
	}
}

func init() {
	testApp = app.New()
	testApp.Secret = stringutil.Random(32)
	testApp.Handle("^/hello$", func(ctx *app.Context) {
		ctx.Header().Add("X-Hello", "World")
		ctx.Header().Add("X-Number", "42")
		ctx.WriteString("hello world")
	})
	testApp.Handle("^/empty$", func(ctx *app.Context) {})
	testApp.Handle("^/echo$", func(ctx *app.Context) {
		if ctx.R.Method == "POST" {
			data, err := ioutil.ReadAll(ctx.R.Body)
			if err != nil {
				panic(err)
			}
			ctx.Write(data)
		}
	})
	testApp.Handle("^/echo-form$", func(ctx *app.Context) {
		if err := ctx.R.ParseForm(); err != nil {
			panic(err)
		}
		var values url.Values
		if ctx.R.Method == "POST" || ctx.R.Method == "PUT" {
			values = ctx.R.PostForm
		} else {
			values = ctx.R.Form
		}
		keys := generic.Keys(values).([]string)
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(ctx, "%s=%s\n", k, values.Get(k))
		}
	})
	testApp.Handle("^/invalid-write-header$", func(ctx *app.Context) {
		ctx.WriteHeader(0)
	})
	testApp.Handle("^/multiple-write-header$", func(ctx *app.Context) {
		ctx.WriteHeader(200)
		ctx.WriteHeader(300)
	})
}
