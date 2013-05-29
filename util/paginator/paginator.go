package paginator

import (
	"html/template"
	"strconv"
)

var (
	DefaultOffset = 5
)

type Paginator struct {
	Base    string
	Current int
	Count   int
	Offset  int
	Pager   Pager
}

func (p *Paginator) pageHref(page int) string {
	return p.Pager.Href(p.Base, page)
}

func (p *Paginator) appendNode(parent, cur *Node, page, flags int) {
	node := p.Pager.Node(cur, page, flags)
	if node != nil {
		parent.Children = append(parent.Children, node)
	}
}

func (p *Paginator) Render() template.HTML {
	root := p.Pager.Root()
	parent := root
	for parent.Children != nil {
		parent = parent.Children[len(parent.Children)-1]
	}
	flags := PREVIOUS | DISABLED
	prev := &Node{Tag: "a", Attributes: Attributes{}}
	if p.Current > 1 {
		prev.Attributes["href"] = p.pageHref(p.Current - 1)
		flags &= ^DISABLED
	}
	p.appendNode(parent, prev, p.Current-1, flags)
	left := p.Current - p.Offset
	if left < 1 {
		left = 1
	}
	if left > 1 {
		p.appendNode(parent, &Node{Tag: "a"}, 0, SEPARATOR)
	}
	for ; left < p.Current; left++ {
		node := &Node{Tag: "a", Head: strconv.Itoa(left), Attributes: Attributes{"href": p.pageHref(left)}}
		p.appendNode(parent, node, left, 0)
	}
	p.appendNode(parent, &Node{Tag: "a", Head: strconv.Itoa(p.Current)}, p.Current, CURRENT)
	right := p.Current + p.Offset
	if right > p.Count {
		right = p.Count
	}
	for jj := p.Current + 1; jj <= right; jj++ {
		node := &Node{Tag: "a", Head: strconv.Itoa(jj), Attributes: Attributes{"href": p.pageHref(jj)}}
		p.appendNode(parent, node, jj, 0)
	}
	if right < p.Count {
		p.appendNode(parent, &Node{Tag: "a"}, 0, SEPARATOR)
	}
	flags = NEXT | DISABLED
	next := &Node{Tag: "a", Attributes: Attributes{}}
	if p.Current < p.Count {
		next.Attributes["href"] = p.pageHref(p.Current + 1)
		flags &= ^DISABLED
	}
	p.appendNode(parent, next, p.Current+1, flags)
	return template.HTML(root.Render())
}

func New(base string, current, count int, pager Pager) *Paginator {
	return &Paginator{
		Base:    base,
		Current: current,
		Count:   count,
		Offset:  DefaultOffset,
		Pager:   pager,
	}
}
