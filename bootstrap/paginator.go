package bootstrap

import (
	"gnd.la/util/paginator"
	"strings"
)

type Pager struct {
	*paginator.SimplePager
	Size      Size
	Alignment Alignment
}

func (p *Pager) Root() *paginator.Node {
	classes := []string{"pagination"}
	switch p.Size {
	case SizeMini:
		classes = append(classes, "pagination-mini")
	case SizeSmall:
		classes = append(classes, "pagination-small")
	case SizeLarge:
		classes = append(classes, "pagination-large")
	}
	switch p.Alignment {
	case AlignmentCenter:
		classes = append(classes, "pagination-centered")
	case AlignmentRight:
		classes = append(classes, "pagination-right")
	}
	return &paginator.Node{
		Tag:        "div",
		Attributes: paginator.Attributes{"class": strings.Join(classes, " ")},
		Children:   []*paginator.Node{&paginator.Node{Tag: "ul"}},
	}
}

func (p *Pager) Node(n *paginator.Node, page int, flags int) *paginator.Node {
	node := p.SimplePager.Node(n, page, flags)
	if flags&paginator.SEPARATOR != 0 || flags&paginator.CURRENT != 0 {
		node.Children[0].Tag = "span"
	}
	return node
}

func NewPaginator(base string, current, count int, next, prev, sep string, s Size, a Alignment, f paginator.Func) *paginator.Paginator {
	pager := &Pager{
		SimplePager: &paginator.SimplePager{
			Wrapper:       "li",
			Next:          next,
			Prev:          prev,
			Separator:     sep,
			CurrentClass:  "active",
			DisabledClass: "disabled",
			Func:          f,
		},
		Size:      s,
		Alignment: a,
	}
	return paginator.New(base, current, count, pager)
}
