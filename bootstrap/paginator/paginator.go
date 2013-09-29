package paginator

import (
	"gnd.la/bootstrap"
	"gnd.la/html"
	"gnd.la/html/paginator"
	"strings"
)

var (
	// Fmt is an alias for gnd.la/html/paginator/Fmt, so
	// users don't have to import it just to use this function.
	Fmt = paginator.Fmt
)

// Paginator represents a paginator which is rendered using
// Bootstrap's markup.
type Paginator struct {
	*paginator.Paginator
	pager *Pager
}

// Pager returns the pager used by this paginator.
func (p *Paginator) Pager() *Pager {
	return p.pager
}

// Size returns the paginator size.
func (p *Paginator) Size() bootstrap.Size {
	return p.pager.Size
}

// SetSize changes the paginator size.
func (p *Paginator) SetSize(s bootstrap.Size) {
	p.pager.Size = s
}

// Alignment returns the paginator alignment.
func (p *Paginator) Alignment() bootstrap.Alignment {
	return p.pager.Alignment
}

// SetAlignment sets the paginator alignment.
func (p *Paginator) SetAlignment(a bootstrap.Alignment) {
	p.pager.Alignment = a
}

// Pager implements the Pager interface and is used by
// the Paginator to render itself.
type Pager struct {
	*paginator.SimplePager
	Size      bootstrap.Size
	Alignment bootstrap.Alignment
}

func (p *Pager) Root() *html.Node {
	classes := []string{"pagination"}
	containerStyles := []string{}
	switch p.Size {
	case bootstrap.ExtraSmall:
		classes = append(classes, "pagination-xs")
	case bootstrap.Small:
		classes = append(classes, "pagination-sm")
	case bootstrap.Large:
		classes = append(classes, "pagination-lg")
	}
	switch p.Alignment {
	case bootstrap.Center:
		containerStyles = append(containerStyles, "text-align: center;")
	case bootstrap.Right:
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

func New(base string, current, count int, next, prev, sep string, f paginator.Func) *Paginator {
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
	}
	return &Paginator{
		Paginator: paginator.New(base, current, count, pager),
		pager:     pager,
	}
}
