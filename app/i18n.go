package app

import (
	"gnd.la/i18n"
	"gnd.la/i18n/table"
)

func (c *Context) Language() string {
	if c.app.languageHandler != nil {
		return c.app.languageHandler(c)
	}
	return c.app.defaultLanguage
}

func (c *Context) TranslationTable() *table.Table {
	if !c.hasTranslations {
		c.translations = table.Get(c.Language())
		c.hasTranslations = true
	}
	return c.translations
}

func (c *Context) T(str string) string {
	return i18n.T(str, c)
}
func (c *Context) Tn(singular string, plural string, n int) string {
	return i18n.Tn(singular, plural, n, c)
}

func (c *Context) Tc(context string, str string) string {
	return i18n.Tc(context, str, c)
}

func (c *Context) Tnc(context string, singular string, plural string, n int) string {
	return i18n.Tnc(context, singular, plural, n, c)
}
