package paginator

import "gnd.la/html"

type PageFlags int

const (
	PageCurrent PageFlags = 1 << iota
	PageDisabled
	PageSeparator
	PagePrevious
	PageNext
)

// Render is the interface implemented by the paginator
// Renderer.
type Renderer interface {
	// Root returns the root node for the paginator. All page
	// Nodes will be added as children of this node.
	Root() *html.Node
	// Node returns the HTML node for the given page number
	// and flags.
	// The returned node must be or contain an "a" node.
	// Each node returned from this function will be added
	// as a child of the returned Root Node
	Node(page int, flags PageFlags) *html.Node
}

// ElementRenderer implements a Renderer which returns
// all HTML nodes inside its RootElement and creates
// each page element by copying its Element attribute
// and adding CurrentClass and/or DisabledClass as
// aproppriate. Note that Element must contain an "a"
// node.
type ElementRenderer struct {
	RootElement   *html.Node
	Element       *html.Node
	CurrentClass  string
	DisabledClass string
}

func (r *ElementRenderer) Root() *html.Node {
	return r.RootElement.Copy()
}

func (r *ElementRenderer) Node(page int, flags PageFlags) *html.Node {
	node := r.Element.Copy()
	if flags&PageCurrent != 0 && r.CurrentClass != "" {
		node.AddClass(r.CurrentClass)
	}
	if flags&PageDisabled != 0 && r.DisabledClass != "" {
		node.AddClass(r.DisabledClass)
	}
	return node
}

var (
	rendererFunc             = defaultRendererFunc
	defaultRenderer Renderer = &ElementRenderer{
		RootElement:   &html.Node{Tag: "ul"},
		Element:       &html.Node{Tag: "li", Children: &html.Node{Tag: "a"}},
		CurrentClass:  "current",
		DisabledClass: "disabled",
	}
)

func defaultRendererFunc() Renderer {
	return defaultRenderer
}

// SetDefaultRenderer sets the function which will return a default
// renderer. The default value returns an ElementRenderer using a ul
// to wrap all the paginator and a li for each page.
func SetDefaultRenderer(f func() Renderer) {
	if f == nil {
		f = defaultRendererFunc
	}
	rendererFunc = f
}

// DefaultRenderer return a new Renderer using the default function.
func DefaultRenderer() Renderer {
	if rendererFunc != nil {
		return rendererFunc()
	}
	return nil
}
