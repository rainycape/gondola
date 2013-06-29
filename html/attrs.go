package html

import (
	"gondola/types"
	"strings"
)

type Attrs map[string]string

func (a Attrs) Add(key, value string) {
	if cur, ok := a[key]; ok {
		a[key] = cur + " " + value
	} else {
		a[key] = value
	}
}

func (a Attrs) Remove(key, value string) {
	if cur, ok := a[key]; ok {
		val := strings.TrimSpace(strings.Replace(cur, value, "", 1))
		if val == "" {
			delete(a, key)
		} else {
			a[key] = val
		}
	}
}

func (n *Node) Attr(name string) string {
	if n.Attrs != nil {
		return n.Attrs[name]
	}
	return ""
}

func (n *Node) DelAttr(name string) *Node {
	if n.Attrs != nil {
		delete(n.Attrs, name)
	}
	return n
}

func (n *Node) SetAttr(name string, value interface{}) *Node {
	if n.Attrs == nil {
		n.Attrs = Attrs{}
	}
	n.Attrs[name] = types.ToString(value)
	return n
}

func (n *Node) AddAttr(name string, value interface{}) *Node {
	if n.Attrs == nil {
		n.Attrs = Attrs{}
	}
	n.Attrs.Add(name, types.ToString(value))
	return n
}

func (n *Node) RemoveAttr(name string, value interface{}) *Node {
	if n.Attrs != nil {
		n.Attrs.Remove(name, types.ToString(value))
	}
	return n
}

func (n *Node) Enable(name string) *Node {
	return n.SetAttr(name, name)
}

func (n *Node) Disable(name string) *Node {
	return n.DelAttr(name)
}

func (n *Node) AddClass(cls string) *Node {
	return n.AddAttr("class", cls)
}

func (n *Node) RemoveClass(cls string) *Node {
	return n.RemoveAttr("class", cls)
}
