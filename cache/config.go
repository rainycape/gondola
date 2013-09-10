package cache

import (
	"fmt"
	"gondola/cache/driver"
	"net/url"
	"strings"
)

type config struct {
	Driver  string
	Value   string
	Options driver.Options
}

func (c *config) Get(key string) string {
	if c.Options != nil {
		return c.Options[key]
	}
	return ""
}

func (c *config) String() string {
	options := ""
	if len(c.Options) > 0 {
		vals := url.Values{}
		for k, v := range c.Options {
			vals.Set(k, v)
		}
		options = "?" + vals.Encode()
	}
	return fmt.Sprintf("%s:%s%s", c.Driver, c.Value, options)
}

func parseConfig(s string) (*config, error) {
	p := strings.Index(s, "://")
	if p < 0 {
		return nil, fmt.Errorf("invalid cache config %q", s)
	}
	drv, value := s[:p], s[p+3:]
	options := driver.Options{}
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
	return &config{
		Driver:  drv,
		Value:   value,
		Options: options,
	}, nil
}
