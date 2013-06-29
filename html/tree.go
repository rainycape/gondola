package html

func (n *Node) Append(node *Node) *Node {
	if n.Children == nil {
		n.Children = node
	} else {
		cur := n.Children
		for cur.Next != nil {
			cur = cur.Next
		}
		cur.Next = node
	}
	return n
}

func (n *Node) Prepend(node *Node) *Node {
	node.Next = n.Children
	n.Children = node
	return n
}

func (n *Node) AppendTo(node *Node) *Node {
	return node.Append(n)
}

func (n *Node) PrependTo(node *Node) *Node {
	return node.Prepend(n)
}
