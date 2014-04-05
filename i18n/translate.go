package i18n

// T returns the given string translated into the language
// returned by lang.
func T(lang Languager, str string) string {
	return Tc(lang, "", str)
}

// Tn translates the given string into the language returned
// by lang. The string will have different forms for singular
// and plural forms. The chosen form will depend on the n
// parameter and the target language. If there's no translation,
// the singular form will be returned iff n = 1.
func Tn(lang Languager, singular string, plural string, n int) string {
	return Tnc(lang, "", singular, plural, n)
}

// Tc works like T, but accepts an additional context argument, to allow
// differentiating strings with the same singular form but different
// translation depending on the context.
func Tc(lang Languager, context string, str string) string {
	if translations := getTable(lang); translations != nil {
		return translations.Singular(context, str)
	}
	return str
}

// Tnc works like Tn, but accepts an additional context argument, to allow
// differentiating strings with the same singular form but different
// translation depending on the context. See the documentation for Tn for
// information about which form (singular or plural) is chosen.
func Tnc(lang Languager, context string, singular string, plural string, n int) string {
	if translations := getTable(lang); translations != nil {
		return translations.Plural(context, singular, plural, n)
	}
	if n == 1 {
		return singular
	}
	return plural
}
