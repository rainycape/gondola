package paginator

import (
	"bytes"
)

type Attributes map[string]string

type Node struct {
	Tag        string
	Head       string
	Tail       string
	Attributes Attributes
	Children   []*Node
}

func (n *Node) Render() string {
	var buf bytes.Buffer
	n.render(&buf)
	return buf.String()
}

func (n *Node) render(buf *bytes.Buffer) {
	if n.Tag != "" {
		buf.WriteByte('<')
		buf.WriteString(n.Tag)
		for k, v := range n.Attributes {
			buf.WriteByte(' ')
			buf.WriteString(k)
			buf.WriteString("=\"")
			buf.WriteString(v)
			buf.WriteByte('"')
		}
		buf.WriteByte('>')
	}
	buf.WriteString(n.Head)
	for _, c := range n.Children {
		c.render(buf)
	}
	buf.WriteString(n.Tail)
	if n.Tag != "" {
		buf.WriteString("</")
		buf.WriteString(n.Tag)
		buf.WriteByte('>')
	}
}
