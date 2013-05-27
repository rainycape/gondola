package cache

import (
	"fmt"
	"gondola/cache/driver"
	"net/url"
	"strings"
)

type Config struct {
	Driver  string
	Value   string
	Options driver.Options
}

func (c *Config) Get(key string) string {
	if c.Options != nil {
		return c.Options[key]
	}
	return ""
}

func (c *Config) String() string {
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

func ParseConfig(config string) (*Config, error) {
	p := strings.Index(config, "://")
	if p < 0 {
		return nil, fmt.Errorf("Invalid cache config %q", config)
	}
	drv, value := config[:p], config[p+3:]
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
	return &Config{
		Driver:  drv,
		Value:   value,
		Options: options,
	}, nil
}

func MustParseConfig(config string) *Config {
	cfg, err := ParseConfig(config)
	if err != nil {
		panic(err)
	}
	return cfg
}
