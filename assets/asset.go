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
	AssetName() string
	AssetPosition() Position
	AssetCondition() *Condition
	AssetTag() string
	AssetClosed() bool
	AssetAttributes() Attributes
	AssetHTML() string
}

func Parse(m Manager, name string, names []string, o Options) ([]Asset, error) {
	parser := parsers[name]
	if parser == nil {
		return nil, fmt.Errorf("unknown asset type %s", name)
	}
	// Check for debug and !debug assets
	if (m.Debug() && o.NoDebug()) || (!m.Debug() && o.Debug()) {
		return nil, nil
	}
	assets, err := parser(m, names, o)
	if err != nil {
		return nil, fmt.Errorf("error parsing asset %q: %s", name, err)
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

func singleParser(parser AssetParser) AssetParser {
	return func(m Manager, names []string, options Options) ([]Asset, error) {
		if len(names) != 1 {
			return nil, fmt.Errorf("this asset accepts only one argument (%d given)", len(names))
		}
		return parser(m, names, options)
	}
}

func Register(name string, parser AssetParser) {
	parsers[name] = parser
}
