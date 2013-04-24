package assets

import (
	"fmt"
)

type Position int

const (
	Top Position = 1 + iota
	Bottom
)

type Options map[string]string

func (o Options) BoolOpt(key string) bool {
	_, ok := o[key]
	return ok
}

func (o Options) StringOpt(key string) string {
	return o[key]
}

func (o Options) Debug() bool {
	return o.BoolOpt("debug")
}

func (o Options) NoDebug() bool {
	return o.BoolOpt("!debug")
}

var (
	parsers = map[string]AssetParser{}
)

type AssetParser func(m Manager, names []string, options Options) ([]Asset, error)

type Asset interface {
	Name() string
	Position() Position
	Condition() Condition
	ConditionVersion() int
	Tag() string
	Closed() bool
	Attributes() Attributes
	HTML() string
}

func Parse(m Manager, name string, names []string, o Options) ([]Asset, error) {
	parser := parsers[name]
	if parser == nil {
		return nil, fmt.Errorf("Unknown asset type %s", name)
	}
	// Check for debug and !debug assets
	if (m.Debug() && o.NoDebug()) || (!m.Debug() && o.Debug()) {
		return nil, nil
	}
	assets, err := parser(m, names, o)
	if err != nil {
		return nil, err
	}
	return assets, nil
}

func Register(name string, parser AssetParser) {
	parsers[name] = parser
}
