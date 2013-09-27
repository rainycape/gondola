package config

import (
	"fmt"
	"net/url"
	"strings"
)

// Options is a conveniency type for representing
// the options specified in a configuration pseudo-url.
type Options map[string]string

// Get returns the value for the given option, or the
// empty string if no such option was specified.
func (o Options) Get(key string) string {
	return o[key]
}

// URL represents a pseudo URL which is used to specify
// a configuration e.g. postgres://dbname=foo user=bar password=baz.
// Config URLs are parsed using the following algorithm:
//	- Anything before the :// is parsed as Sheme
//	- The part from the :// until the end or the first ? is parsed as Value
//	- Anything after the ? is parsed as a query string and stored in Options
type URL struct {
	Scheme  string
	Value   string
	Options Options
}

// Get returns the value for the given option, or
// the empty string if there are no options or
// this key wasn't provided.
func (u *URL) Get(key string) string {
	if u.Options != nil {
		return u.Options.Get(key)
	}
	return ""
}

func ParseURL(s string) (*URL, error) {
	p := strings.Index(s, "://")
	if p < 0 {
		return nil, fmt.Errorf("invalid config URL %q", s)
	}
	scheme, value := s[:p], s[p+3:]
	options := Options{}
	if q := strings.Index(value, "?"); q >= 0 {
		val, err := url.ParseQuery(value[q+1:])
		if err != nil {
			return nil, err
		}
		for k := range val {
			options[k] = val.Get(k)
		}
		value = value[:q]
	}
	return &URL{
		Scheme:  scheme,
		Value:   value,
		Options: options,
	}, nil
}
