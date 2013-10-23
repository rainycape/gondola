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

func Sprintf(format string, lang Languager, args ...interface{}) string {
	format = T(format, lang)
	return sprintf(lang, format, args)
}

func Sprintfc(ctx string, format string, lang Languager, args ...interface{}) string {
	format = Tc(ctx, format, lang)
	return sprintf(lang, format, args)
}

func Sprintfn(singular string, plural string, n int, lang Languager, args ...interface{}) string {
	format := Tn(singular, plural, n, lang)
	return sprintf(lang, format, args)
}

func Sprintfnc(ctx string, singular string, plural string, n int, lang Languager, args ...interface{}) string {
	format := Tnc(ctx, singular, plural, n, lang)
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
