package paginator

import (
	"fmt"
	"html/template"
	"strconv"

	"gnd.la/html"
)

var (
	DefaultOffset = 5
)

const (
	defaultPrevious  = "&laquo;"
	defaultSeparator = "&hellip;"
	defaultNext      = "&raquo;"
)

type Flags int

const (
	// FlagNoPrevNext removes the links to the previous and
	// next page at the start and the end of the paginator.
	FlagNoPrevNext = 1 << iota
	// FlagNoBoundaries does not show the paginator boundaries.
	// When showing the boundaries the first and the last page are
	// always shown.
	FlagNoBoundaries
)

type Paginator struct {
	// Count is the total number of pages
	Count int
	// Current represents the page number (1-indexed)
	// to be rendered as the current page
	Current int
	// Offset is the number of pages shown at each
	// side of the current one. Its default value
	// is copied from DefaultOffset when a Paginator
	// is created via New.
	Offset int
	// Flags control several aspects of the rendering.
	// See the Flags type constants for the available ones.
	Flags Flags
	// Labels used in the paginator. If empty, their values
	// will default to &laquo;, &hellip; and &raquo;, respectivelly.
	Previous, Separator, Next string
	// Interfaces used to render the HTML
	Pager    Pager
	Renderer Renderer
}

func (p *Paginator) appendNode(parent *html.Node, page int, flags PageFlags) {
	if page == p.Current {
		flags |= PageCurrent
	}
	node := p.Renderer.Node(page, flags)
	var text string
	switch {
	case flags&PageSeparator != 0:
		text = p.Separator
		if text == "" {
			text = defaultSeparator
		}
	case flags&PagePrevious != 0:
		text = p.Previous
		if text == "" {
			text = defaultPrevious
		}
	case flags&PageNext != 0:
		text = p.Next
		if text == "" {
			text = defaultNext
		}
	default:
		text = strconv.Itoa(page)
	}
	anchor := node.Find(html.TypeAny, "a", nil)
	if anchor == nil {
		panic(fmt.Errorf("no anchor found in ElementRenderer's element %s", node))
	}
	anchor.Children = html.Text(text)
	if page > 0 && page != p.Current && page <= p.Count {
		anchor.SetAttr("href", p.Pager.URL(page))
	}
	parent.AppendChild(node)
}

// Render renders the Paginator as HTML. It's usually
// called from a template e.g.
//
//  {{ with .Paginator }}
//	{{ .Render }}
//  {{ end }}
func (p *Paginator) Render() template.HTML {
	if p.Renderer == nil {
		p.Renderer = DefaultRenderer()
	}
	root := p.Renderer.Root()
	parent := root
	for parent.Children != nil {
		parent = parent.LastChild()
	}
	var flags PageFlags
	if p.Flags&FlagNoPrevNext == 0 {
		flags = PagePrevious
		if p.Current <= 1 {
			flags |= PageDisabled
		}
		p.appendNode(parent, p.Current-1, flags)
	}
	left := p.Current - p.Offset
	if left < 1 {
		left = 1
	}
	if left > 1 && p.Flags&FlagNoBoundaries == 0 {
		p.appendNode(parent, 1, 0)
		if left > 2 {
			p.appendNode(parent, -1, PageSeparator|PageDisabled)
		}
	}
	for ; left < p.Current; left++ {
		p.appendNode(parent, left, 0)
	}
	p.appendNode(parent, p.Current, PageCurrent)
	right := p.Current + p.Offset
	if right > p.Count {
		right = p.Count
	}
	for jj := p.Current + 1; jj <= right; jj++ {
		p.appendNode(parent, jj, 0)
	}
	if right < p.Count && p.Flags&FlagNoBoundaries == 0 {
		if right < p.Count-1 {
			p.appendNode(parent, -1, PageSeparator|PageDisabled)
		}
		p.appendNode(parent, p.Count, 0)
	}
	if p.Flags&FlagNoPrevNext == 0 {
		flags = PageNext
		if p.Current == p.Count {
			flags |= PageDisabled
		}
		p.appendNode(parent, p.Current+1, flags)
	}
	return root.HTML()
}

// New returns a new Paginator with the given page count,
// current page and Pager to obtain the URL for each page.
// The returned Paginator will use the Renderer returned by
// DefaultRenderer, but its Renderer attribute might be
// modified at any time.
func New(count int, current int, pager Pager) *Paginator {
	return &Paginator{
		Count:    count,
		Current:  current,
		Offset:   DefaultOffset,
		Pager:    pager,
		Renderer: DefaultRenderer(),
	}
}
