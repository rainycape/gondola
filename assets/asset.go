package assets

import (
	"fmt"
	"gnd.la/log"
)

type Position int

const (
	Top Position = 1 + iota
	Bottom
)

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
	if o.BoolOpt("compile", m) {
		cassets, err := Compile(m, assets, o)
		if err == nil {
			assets = cassets
		} else {
			log.Errorf("Error compiling assets %s:%s: %s. Using uncompiled.", name, names, err)
		}
	}
	return assets, nil
}

func Register(name string, parser AssetParser) {
	parsers[name] = parser
}
