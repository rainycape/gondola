package table

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	registry = make(map[string]string)
	decoded  = make(map[string]*Table)
	cache    = make(map[string]*Table)
	mu       sync.RWMutex
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
func Register(lang string, data string) {
	if err := register(lang, data); err != nil {
		panic(err)
	}
}

func languageKey(k string) string {
	return strings.ToUpper(strings.Replace(k, "_", "-", -1))
}

func register(lang string, data string) error {
	if len(lang) != 2 && len(lang) != 5 {
		return fmt.Errorf("invalid language code %q, please see the documentation for Register()", lang)
	}
	if len(data) == 0 {
		return fmt.Errorf("invalid table for language %q, no data", lang)
	}
	key := languageKey(lang)
	if prev := registry[key]; prev == "" {
		registry[key] = data
	} else {
		prevt, err := Decode(prev)
		if err != nil {
			return err
		}
		cur, err := Decode(data)
		if err != nil {
			return err
		}
		if err := prevt.Update(cur); err != nil {
			return err
		}
		compressed, err := prevt.Encode()
		if err != nil {
			return err
		}
		registry[key] = compressed
	}
	return nil
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

func Get(lang string) *Table {
	mu.RLock()
	t, ok := cache[lang]
	mu.RUnlock()
	if ok {
		return t
	}
	key := languageKey(lang)
	t = getSkippingCache(key)
	if t == nil {
		// Check if any of the registered tables are suitable
		// for this language
		if len(key) == 2 {
			for k := range registry {
				if key == k[:2] {
					t = getSkippingCache(k)
					break
				}
			}
		} else if len(key) == 5 {
			sk := key[:2]
			for k := range registry {
				if sk == k {
					t = getSkippingCache(k)
					break
				}
				if sk == k[:2] {
					t = getSkippingCache(k)
				}
			}
		}
	}
	mu.Lock()
	cache[lang] = t
	mu.Unlock()
	// Try to decompress
	return t
}

func getSkippingCache(key string) *Table {
	if t := decoded[key]; t != nil {
		return t
	}
	if d := registry[key]; d != "" {
		t, err := Decode(d)
		if err != nil {
			panic(err)
		}
		mu.Lock()
		decoded[key] = t
		mu.Unlock()
		return t
	}
	return nil
}
