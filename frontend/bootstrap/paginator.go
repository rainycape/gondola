package bootstrap

import (
	"gnd.la/html"
	"gnd.la/html/paginator"
)

// PaginatorRenderer implements a gnd.la/html/paginator
// Renderer using bootstrap. The size attribute might be used
// to alter the size of the pager. Note that currently bootstrap
// paginators might be only Medium (the default), Small or Large.
type PaginatorRenderer struct {
	Size Size
}

func (r *PaginatorRenderer) Root() *html.Node {
	ul := &html.Node{Tag: "ul"}
	ul.AddClass("pagination")
	if r.Size != Medium {
		ul.AddClass("pagination-" + r.Size.String())
	}
	return ul
}

func (r *PaginatorRenderer) Node(page int, flags paginator.PageFlags) *html.Node {
	li := &html.Node{Tag: "li", Children: &html.Node{Tag: "a"}}
	if flags&paginator.PageDisabled != 0 {
		li.AddClass("disabled")
	}
	if flags&paginator.PageCurrent != 0 {
		li.AddClass("active")
	}
	return li
}

func init() {
	paginator.SetDefaultRenderer(func() paginator.Renderer {
		return &PaginatorRenderer{}
	})
}
