package table

import (
	"fmt"
	"sort"
	"strings"
)

var (
	registry = map[string]*TranslationTable{}
)

// Register registers a new binary table for the given language.
// Keep in mind that Register is meant to be called from init and
// it's not thread safe. The first parameter must be either an
// ISO-639-1 language code, like "es" or "us", or either an
// ISO-639-1/ISO-3166-1-alpha2 combination, like "es_ES", "en_US"
// or "en_GB". Note that internally all codes are translated to
// uppercase and dashes are translated to underscores. This means
// that the languages "ES-ES", "es_es" and "es_ES" are equivalent.
// The second parameter is a compressed language table. If there's already
// a table registered for the given language, it will be updated with
// the new table, adding or updating entries as required.
func Register(lang string, data []byte) {
	if len(lang) != 2 && len(lang) != 5 {
		panic(fmt.Errorf("invalid language code %q, please see the documentation for Register()", lang))
	}
	if len(data) == 0 {
		panic(fmt.Errorf("invalid table for language %q, no data", lang))
	}
	if data[0] != BZIP2 {
		panic(fmt.Errorf("invalid table compression %d for language %q", data[0], lang))
	}
}

func Registered() []string {
	// Return entries in the xx_YY format
	entries := make([]string, len(registry))
	ii := 0
	for k := range registry {
		if len(k) == 2 {
			// xx
			entries[ii] = strings.ToLower(k)
		} else {
			// must be xx_YY
			entries[ii] = strings.ToLower(k[:2]) + "_" + strings.ToUpper(k[3:])
		}
	}
	sort.Strings(entries)
	return entries
}

func Table(lang string) *TranslationTable {
	t := registry[lang]
	if t == nil {
	}
	return t
}
