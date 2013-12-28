package assets

import (
	"fmt"
	"gnd.la/util/textutil"
	"strings"
)

type Options map[string]string

func ParseOptions(options string) (Options, error) {
	values, err := textutil.SplitFields(options, ",")
	if err != nil {
		return nil, fmt.Errorf("error parsing asset options: %s", err)
	}
	opts := make(Options)
	for _, v := range values {
		eq := strings.IndexByte(v, '=')
		if eq < 0 {
			opts[v] = ""
		} else {
			opts[v[:eq]] = v[eq+1:]
		}
	}
	return opts, nil
}

func (o Options) BoolOpt(key string) bool {
	_, ok := o[key]
	return ok
}

func (o Options) StringOpt(key string) string {
	return o[key]
}

func (o Options) String() string {
	var values []string
	for k, v := range o {
		values = append(values, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(values, ",")
}

// Common options

func (o Options) Debug() bool {
	return o.BoolOpt("debug")
}

func (o Options) NoDebug() bool {
	return o.BoolOpt("nodebug")
}

func (o Options) Top() bool {
	return o.BoolOpt("top")
}

func (o Options) Async() bool {
	return o.BoolOpt("async")
}

func (o Options) Bundle() bool {
	return o.BoolOpt("bundle")
}

func (o Options) Cdn() bool {
	return o.BoolOpt("cdn")
}
