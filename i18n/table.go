package i18n

import (
	"gnd.la/i18n/table"
)

// Tabler is the interface implemented by types
// which instead of returning a language code, can
// store and return a translation table, resulting in
// better performance. All functions in the i18n package
// which accept a Languager will check if the received
// object implements Tabler.
type Tabler interface {
	TranslationTable() *table.Table
}

func getTable(lang Languager) *table.Table {
	if lang == nil {
		return nil
	}
	if tabler, ok := lang.(Tabler); ok {
		return tabler.TranslationTable()
	}
	return table.Get(lang.Language())
}
