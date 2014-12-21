package html

import (
	"bytes"
	"html/template"
	"strings"
)

type Node struct {
	Type     Type
	Tag      string
	Attrs    Attrs
	Content  string
	Next     *Node
	Children *Node
	Open     bool
}

func (n *Node) NChildren() int {
	ii := 0
	for c := n.Children; c != nil; c = c.Next {
		ii++
	}
	return ii
}

func (n *Node) AppendChild(c *Node) {
	last := n.LastChild()
	if last != nil {
		last.Next = c
	} else {
		n.Children = c
	}
}

func (n *Node) LastChild() *Node {
	c := n.Children
	if c != nil {
		for {
			if c.Next == nil {
				return c
			}
			c = c.Next
		}
	}
	return nil
}

func (n *Node) match(typ Type, tag string, attrs Attrs) bool {
	if n != nil {
		if typ == TypeAny || typ == n.Type {
			if tag == "" || strings.ToLower(tag) == strings.ToLower(n.Tag) {
				return attrs == nil || attrs.Equal(n.Attrs)
			}
		}
	}
	return false
}

// Find returns the first Node (including the receiver) which
// matches the provided query parameters. A Node matches if
// all the following conditions are true.
//
//  typ == TypeAny || n.Type == typ
//  tag == "" || n.Tag == tag (case insensitive)
//  attrs == nil || attrs.Equal(n.Attrs)
//
// The nodes are iterated following the Next node and then
// the first Children Node.
func (n *Node) Find(typ Type, tag string, attrs Attrs) *Node {
	if n.match(typ, tag, attrs) {
		return n
	}
	for nn := n.Next; nn != nil; nn = nn.Next {
		if nn.match(typ, tag, attrs) {
			return nn
		}
	}
	if n.Children != nil {
		return n.Children.Find(typ, tag, attrs)
	}
	return nil
}

// Copy returns a deep copy of the node.
func (n *Node) Copy() *Node {
	if n == nil {
		return nil
	}
	cpy := *n
	cpy.Attrs = make(Attrs, len(n.Attrs))
	for k, v := range n.Attrs {
		cpy.Attrs[k] = v
	}
	cpy.Children = cpy.Children.Copy()
	cpy.Next = cpy.Next.Copy()
	return &cpy
}

func (n *Node) Render(buf *bytes.Buffer) {
	n.writeToStringWriter(buf)
}

func (n *Node) String() string {
	var buf bytes.Buffer
	n.writeToStringWriter(&buf)
	return buf.String()
}

func (n *Node) HTML() template.HTML {
	return template.HTML(n.String())
}
