package paginator

import (
	"gnd.la/html"
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

func (p *Paginator) appendNode(parent, cur *html.Node, page, flags int) {
	node := p.Pager.Node(cur, page, flags)
	if node != nil {
		parent.AppendChild(node)
	}
}

func (p *Paginator) Render() template.HTML {
	root := p.Pager.Root()
	parent := root
	for parent.Children != nil {
		parent = parent.LastChild()
	}
	flags := PREVIOUS | DISABLED
	prev := &html.Node{Tag: "a", Attrs: html.Attrs{}}
	if p.Current > 1 {
		prev.Attrs["href"] = p.pageHref(p.Current - 1)
		flags &= ^DISABLED
	}
	p.appendNode(parent, prev, p.Current-1, flags)
	left := p.Current - p.Offset
	if left < 1 {
		left = 1
	}
	if left > 1 {
		p.appendNode(parent, &html.Node{Tag: "a"}, 0, SEPARATOR)
	}
	for ; left < p.Current; left++ {
		node := &html.Node{Tag: "a", Children: html.Text(strconv.Itoa(left)), Attrs: html.Attrs{"href": p.pageHref(left)}}
		p.appendNode(parent, node, left, 0)
	}
	p.appendNode(parent, &html.Node{Tag: "a", Children: html.Text(strconv.Itoa(p.Current))}, p.Current, CURRENT)
	right := p.Current + p.Offset
	if right > p.Count {
		right = p.Count
	}
	for jj := p.Current + 1; jj <= right; jj++ {
		node := &html.Node{Tag: "a", Children: html.Text(strconv.Itoa(jj)), Attrs: html.Attrs{"href": p.pageHref(jj)}}
		p.appendNode(parent, node, jj, 0)
	}
	if right < p.Count {
		p.appendNode(parent, &html.Node{Tag: "a"}, 0, SEPARATOR)
	}
	flags = NEXT | DISABLED
	next := &html.Node{Tag: "a", Attrs: html.Attrs{}}
	if p.Current < p.Count {
		next.Attrs["href"] = p.pageHref(p.Current + 1)
		flags &= ^DISABLED
	}
	p.appendNode(parent, next, p.Current+1, flags)
	return root.HTML()
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
