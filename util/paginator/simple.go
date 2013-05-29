package paginator

import (
	"strings"
)

type SimplePager struct {
	Tag            string
	Wrapper        string
	Next           string
	Prev           string
	Separator      string
	NextClass      string
	PrevClass      string
	CurrentClass   string
	DisabledClass  string
	SeparatorClass string
	Func           Func
}

func (p *SimplePager) Root() *Node {
	return &Node{Tag: p.Tag}
}

func (p *SimplePager) Href(base string, page int) string {
	return p.Func(base, page)
}

func (p *SimplePager) Node(n *Node, page int, flags int) *Node {
	var classes []string
	if flags&NEXT != 0 {
		if p.NextClass != "" {
			classes = append(classes, p.NextClass)
		}
		n.Head = p.Next
	} else if flags&PREVIOUS != 0 {
		if p.PrevClass != "" {
			classes = append(classes, p.PrevClass)
		}
		n.Head = p.Prev
	} else if flags&SEPARATOR != 0 {
		if p.SeparatorClass != "" {
			classes = append(classes, p.SeparatorClass)
		}
		n.Head = p.Separator
	}
	if flags&CURRENT != 0 {
		if p.CurrentClass != "" {
			classes = append(classes, p.CurrentClass)
		}
	} else if flags&DISABLED != 0 {
		if p.DisabledClass != "" {
			classes = append(classes, p.DisabledClass)
		}
	}
	if p.Wrapper != "" {
		var attrs Attributes
		if classes != nil {
			attrs = Attributes{"class": strings.Join(classes, " ")}
		}
		return &Node{Tag: p.Wrapper, Attributes: attrs, Children: []*Node{n}}
	}
	if classes != nil {
		if n.Attributes == nil {
			n.Attributes = Attributes{}
		}
		n.Attributes["class"] = strings.Join(classes, " ")
	}
	return n
}

func NewSimple(base string, current, count int, next, prev, sep string, f Func) *Paginator {
	pager := &SimplePager{
		Tag:       "div",
		Next:      next,
		Prev:      prev,
		Separator: sep,
		Func:      f,
	}
	return New(base, current, count, pager)
}
