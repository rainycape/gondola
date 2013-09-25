package html

import (
	"bytes"
	"html/template"
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
