package i18n

import (
	"fmt"
)

func placeholders(s string) (int, bool) {
	reordered := false
	p := 0
	t := len(s) - 1
	for ii := 0; ii < t; ii++ {
		if s[ii] == '%' {
			if s[ii+1] == '%' {
				ii++
				continue
			}
			if s[ii+1] == '[' {
				reordered = true
			}
			p++
		}
	}
	return p, reordered
}

func Sprintf(lang Languager, format string, args ...interface{}) string {
	format = T(lang, format)
	return sprintf(lang, format, args)
}

func Sprintfc(lang Languager, ctx string, format string, args ...interface{}) string {
	format = Tc(lang, ctx, format)
	return sprintf(lang, format, args)
}

func Sprintfn(lang Languager, singular string, plural string, n int, args ...interface{}) string {
	format := Tn(lang, singular, plural, n)
	return sprintf(lang, format, args)
}

func Sprintfnc(lang Languager, ctx string, singular string, plural string, n int, args ...interface{}) string {
	format := Tnc(lang, ctx, singular, plural, n)
	return sprintf(lang, format, args)
}

func sprintf(lang Languager, format string, args []interface{}) string {
	for ii, v := range args {
		if t, ok := v.(TranslatableString); ok {
			args[ii] = t.TranslatedString(lang)
		}
	}
	if c, reordered := placeholders(format); !reordered && c < len(args) {
		args = args[:c]
	}
	return fmt.Sprintf(format, args...)
}
