package bootstrap

import (
	"gnd.la/html"
	"gnd.la/html/paginator"
	"strings"
)

type Pager struct {
	*paginator.SimplePager
	Size      Size
	Alignment Alignment
}

func (p *Pager) Root() *html.Node {
	classes := []string{"pagination"}
	containerStyles := []string{}
	switch p.Size {
	case SizeExtraSmall:
		classes = append(classes, "pagination-xs")
	case SizeSmall:
		classes = append(classes, "pagination-sm")
	case SizeLarge:
		classes = append(classes, "pagination-lg")
	}
	switch p.Alignment {
	case AlignmentCenter:
		containerStyles = append(containerStyles, "text-align: center;")
	case AlignmentRight:
		containerStyles = append(containerStyles, "text-align: right;")
	}
	root := &html.Node{
		Tag:   "ul",
		Attrs: html.Attrs{"class": strings.Join(classes, " ")},
	}
	if len(containerStyles) > 0 {
		return &html.Node{
			Tag:      "div",
			Attrs:    html.Attrs{"style": strings.Join(containerStyles, " ")},
			Children: root,
		}
	}
	return root
}

func (p *Pager) Node(n *html.Node, page int, flags int) *html.Node {
	node := p.SimplePager.Node(n, page, flags)
	if flags&paginator.SEPARATOR != 0 || flags&paginator.CURRENT != 0 && node.Children != nil {
		node.Children.Tag = "span"
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
