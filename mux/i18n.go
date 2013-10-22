package mux

import (
	"gnd.la/i18n"
)

func (c *Context) Language() string {
	if c.mux.languageHandler != nil {
		return c.mux.languageHandler(c)
	}
	return c.mux.defaultLanguage
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
