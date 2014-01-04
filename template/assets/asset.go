package assets

import (
	"fmt"
	"io/ioutil"
	"strings"
)

type Position int

const (
	None Position = iota
	Top
	Bottom
)

func (p Position) String() string {
	switch p {
	case None:
		return "None"
	case Top:
		return "Top"
	case Bottom:
		return "Bottom"
	}
	return fmt.Sprintf("invalid position %d", p)
}

type Type int

const (
	TypeOther Type = iota
	TypeCSS
	TypeJavascript
)

func (t Type) String() string {
	switch t {
	case TypeOther:
		return "Other"
	case TypeCSS:
		return "CSS"
	case TypeJavascript:
		return "Javascript"
	}
	return fmt.Sprintf("unknown Type %d", t)
}

func (t Type) Ext() string {
	switch t {
	case TypeCSS:
		return "css"
	case TypeJavascript:
		return "js"
	}
	return ""
}

var (
	parsers = map[string]AssetParser{}
)

type SingleAssetParser func(m *Manager, name string, options Options) ([]*Asset, error)
type AssetParser func(m *Manager, names []string, options Options) ([]*Asset, error)

type Asset struct {
	Name       string
	Type       Type
	Position   Position
	Condition  *Condition
	Attributes Attributes
	HTML       string
}

func (a *Asset) IsRemote() bool {
	name := strings.ToLower(a.Name)
	return strings.HasPrefix(name, "//") || strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://")
}

func (a *Asset) IsHTML() bool {
	return a.HTML != ""
}

type Group struct {
	Manager *Manager
	Assets  []*Asset
	Options Options
}

func (g *Group) Names() []string {
	names := make([]string, len(g.Assets))
	for ii, a := range g.Assets {
		names[ii] = a.Name
	}
	return names

}

func (a *Asset) Code(m *Manager) (string, error) {
	f, _, err := m.Load(a.Name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Parse(m *Manager, name string, names []string, opts Options) (*Group, error) {
	parser := parsers[name]
	if parser == nil {
		return nil, fmt.Errorf("unknown asset type %s", name)
	}
	assets, err := parser(m, names, opts)
	if err != nil {
		return nil, fmt.Errorf("error parsing asset %q: %s", name, err)
	}
	if ifopt := opts["if"]; ifopt != "" {
		cond, err := ParseCondition(ifopt)
		if err != nil {
			return nil, err
		}
		if cond != nil {
			for _, v := range assets {
				if v.Condition == nil {
					v.Condition = cond
				}
			}
		}
	}
	return &Group{
		Manager: m,
		Assets:  assets,
		Options: opts,
	}, nil
}

func SingleParser(parser SingleAssetParser) AssetParser {
	return func(m *Manager, names []string, options Options) ([]*Asset, error) {
		if len(names) != 1 {
			return nil, fmt.Errorf("this asset accepts only one argument (%d given)", len(names))
		}
		return parser(m, names[0], options)
	}
}

func Register(name string, parser AssetParser) {
	parsers[name] = parser
}
