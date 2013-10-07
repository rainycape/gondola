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

// String returns the options encoded as a query string.
func (o Options) String() string {
	var values []string
	for k, v := range o {
		values = append(values, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
	}
	return strings.Join(values, "&")
}

// URL represents a pseudo URL which is used to specify
// a configuration e.g. postgres://dbname=foo user=bar password=baz.
// Systems in Gondola using this configuration type include the
// cache, the ORM and the blobstore.
// Config URLs are parsed using the following algorithm:
//	- Anything before the :// is parsed as Sheme
//	- The part from the :// until the end or the first ? is parsed as Value
//	- Anything after the ? is parsed as a query string and stored in Options,
//	    with the difference that multiple values for the same parameter are
//	    not supported. Only the last one is taken into account.
type URL struct {
	Scheme  string
	Value   string
	Options Options
}

// Parse parses the given string into a configuration URL.
func (u *URL) Parse(s string) error {
	_, err := parseURL(u, s)
	return err
}

// String returns the URL as a string.
func (u *URL) String() string {
	s := fmt.Sprintf("%s://%s", u.Scheme, u.Value)
	if len(u.Options) > 0 {
		s += "?" + u.Options.String()
	}
	return s
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

func parseURL(u *URL, s string) (*URL, error) {
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
		for k, v := range val {
			options[k] = v[len(v)-1]
		}
		value = value[:q]
	}
	if u == nil {
		u = &URL{}
	}
	u.Scheme = scheme
	u.Value = value
	u.Options = options
	return u, nil
}

// ParseURL parses the given string into a *URL, if possible.
func ParseURL(s string) (*URL, error) {
	return parseURL(nil, s)
}

// MustParseURL works like ParseURL, but panics if there's an error.
func MustParseURL(s string) *URL {
	u, err := ParseURL(s)
	if err != nil {
		panic(err)
	}
	return u
}
