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

var (
	parsers = map[string]AssetParser{}
)

type AssetParser func(m Manager, names []string, options map[string]string) ([]Asset, error)

type Asset interface {
	Position() Position
	HTML() template.HTML
}

type CodeAsset interface {
	Asset
	CodeType() CodeType
	Code() string
}

func Parse(m Manager, name string, options map[string]string, names []string) ([]Asset, error) {
	parser := parsers[name]
	if parser == nil {
		return nil, fmt.Errorf("Unknown asset type %s", name)
	}
	return parser(m, names, options)
}

func Register(name string, parser AssetParser) {
	parsers[name] = parser
}
