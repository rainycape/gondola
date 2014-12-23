package app_test

import (
	"strconv"
	"testing"

	"gnd.la/app"
	"gnd.la/app/tester"
)

func TestParameters(t *testing.T) {
	a := app.New()
	a.Handle("/parse-string-param/(?P<string>\\w+)?$", func(ctx *app.Context) {
		str := "default"
		ctx.ParseParamValue("string", &str)
		ctx.WriteString(str)
	})
	a.Handle("/parse-int-param/(?P<number>\\d+)?$", func(ctx *app.Context) {
		val := -1
		ctx.ParseParamValue("number", &val)
		ctx.WriteString(strconv.Itoa(val))
	})
	a.Handle("/parse-int-form-value/$", func(ctx *app.Context) {
		val := -1
		ctx.ParseFormValue("v", &val)
		ctx.WriteString(strconv.Itoa(val))
	})
	a.Handle("/must-parse-int-form-value/$", func(ctx *app.Context) {
		val := -1
		ctx.MustParseFormValue("v", &val)
		ctx.WriteString(strconv.Itoa(val))
	})
	a.Handle("/parse-index-value/(\\d+)?$", func(ctx *app.Context) {
		val := -1
		ctx.ParseIndexValue(0, &val)
		ctx.WriteString(strconv.Itoa(val))
	})
	tester := tester.New(t, a)
	tester.Get("/parse-string-param/", nil).Expect(200).Expect("default")
	tester.Get("/parse-string-param/foo", nil).Expect(200).Expect("foo")
	tester.Get("/parse-int-param/", nil).Expect(200).Expect("-1")
	// -10 does not match the handler
	tester.Get("/parse-int-param/-10", nil).Expect(404)
	tester.Get("/parse-int-param/42", nil).Expect(200).Expect("42")

	tester.Get("/parse-int-form-value/", nil).Expect(200).Expect("-1")
	tester.Get("/parse-int-form-value/", map[string]interface{}{"v": 9000}).Expect(200).Expect("9000")
	tester.Get("/parse-int-form-value/", map[string]interface{}{"v": "not-a-number"}).Expect(200).Expect("-1")

	tester.Get("/must-parse-int-form-value/", nil).Expect(400)
	tester.Get("/must-parse-int-form-value/", map[string]interface{}{"v": 9000}).Expect(200).Expect("9000")
	tester.Get("/must-parse-int-form-value/", map[string]interface{}{"v": "not-a-number"}).Expect(400)

	tester.Get("/parse-index-value/", nil).Expect(200).Expect("-1")
	tester.Get("/parse-index-value/42", nil).Expect(200).Expect("42")
}
