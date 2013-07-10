package html

func container(tag string, children []*Node) *Node {
	n := &Node{
		Type: TAG_NODE,
		Tag:  tag,
	}
	if len(children) > 0 {
		n.Children = children[0]
		for ii, v := range children[:len(children)-1] {
			v.Next = children[ii+1]
		}
	}
	return n
}

func tag(tag string, children *Node) *Node {
	return &Node{
		Type:     TAG_NODE,
		Tag:      tag,
		Children: children,
	}
}

func ttag(tag, text string) *Node {
	var children *Node
	if text != "" {
		children = &Node{
			Type:    TEXT_NODE,
			Content: text,
		}

	}
	return &Node{
		Type:     TAG_NODE,
		Tag:      tag,
		Children: children,
	}
}

func Text(text string) *Node {
	return &Node{
		Type:    TEXT_NODE,
		Content: text,
	}
}

func A(href, text string) *Node {
	return ttag("a", text).SetAttr("href", href)
}

func Abbr(title, text string) *Node {
	return ttag("abbr", text).SetAttr("title", title)
}

func Article(children ...*Node) *Node {
	return container("article", children)
}

func Aside(children ...*Node) *Node {
	return container("aside", children)
}

func B(text string) *Node {
	return ttag("b", text)
}

func Blockquote(text string) *Node {
	return ttag("blockquote", text)
}

func Br() *Node {
	return &Node{
		Type: TAG_NODE,
		Tag:  "br",
		Open: true,
	}
}

func Button(text string) *Node {
	return ttag("button", text)
}

func Caption(text string) *Node {
	return ttag("caption", text)
}

func Div(children ...*Node) *Node {
	return container("div", children)
}

func Em(text string) *Node {
	return ttag("em", text)
}

func P(children ...*Node) *Node {
	return container("p", children)
}

func Small(text string) *Node {
	return ttag("small", text)
}

func Span(text string) *Node {
	return ttag("span", text)
}
