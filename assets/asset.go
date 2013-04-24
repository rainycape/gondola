package assets

import (
	"fmt"
	"html/template"
)

type Position int

const (
	Top Position = 1 + iota
	Bottom
)

type CodeType int

const (
	CodeTypeNone CodeType = iota
	CodeTypeCss
	CodeTypeJavascript
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
	Position() Position
	HTML() template.HTML
}

type CodeAsset interface {
	Asset
	CodeType() CodeType
	Code() string
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
	return parser(m, names, o)
}

func Register(name string, parser AssetParser) {
	parsers[name] = parser
}
