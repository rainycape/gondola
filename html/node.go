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

func (n *Node) String() string {
	var buf bytes.Buffer
	n.writeToStringWriter(&buf)
	return buf.String()
}

func (n *Node) HTML() template.HTML {
	return template.HTML(n.String())
}
