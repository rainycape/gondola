package assets

import (
	"fmt"
	"io/ioutil"
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

var (
	parsers  = map[string]AssetParser{}
	urlAttrs = []string{"href", "src"}
)

type SingleAssetParser func(m *Manager, name string, options Options) ([]*Asset, error)
type AssetParser func(m *Manager, names []string, options Options) ([]*Asset, error)

type Asset struct {
	Manager    *Manager
	Name       string
	Position   Position
	Condition  *Condition
	CodeType   CodeType
	Tag        string
	Closed     bool
	Attributes Attributes
	HTML       string
}

func (a *Asset) Rename(name string) error {
	a.Name = name
	return a.SetURL(a.Manager.URL(name))
}

func (a *Asset) SetURL(url string) error {
	for _, v := range urlAttrs {
		if _, ok := a.Attributes[v]; ok {
			a.Attributes[v] = url
			return nil
		}
		return nil
	}
	return fmt.Errorf("can't set URL on asset %q - doesn't have any ot these attributs %s", a.Name, urlAttrs)
}

type Group struct {
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

func (a *Asset) Code() (string, error) {
	f, _, err := a.Manager.Load(a.Name)
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
	for _, v := range assets {
		v.Manager = m
	}
	return &Group{
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
