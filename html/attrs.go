package html

import (
	"gondola/types"
	"io"
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

func (a Attrs) WriteTo(w io.Writer) (int, error) {
	if sw, ok := w.(stringWriter); ok {
		return a.writeToStringWriter(sw)
	}
	return a.writeTo(w)
}

func (a Attrs) writeTo(w io.Writer) (int, error) {
	t := 0
	for k, v := range a {
		_, err := w.Write([]byte{' '})
		if err != nil {
			return 0, err
		}
		t += 1
		c, err := w.Write([]byte(k))
		if err != nil {
			return 0, err
		}
		t += c
		_, err = w.Write([]byte{'=', '"'})
		if err != nil {
			return 0, err
		}
		t += 1
		c, err = w.Write([]byte(Escape(v)))
		if err != nil {
			return 0, err
		}
		t += c
		_, err = w.Write([]byte{'"'})
		if err != nil {
			return 0, err
		}
	}
	return t, nil
}

func (a Attrs) writeToStringWriter(w stringWriter) (int, error) {
	t := 0
	for k, v := range a {
		_, err := w.WriteString(" ")
		if err != nil {
			return 0, err
		}
		t += 1
		c, err := w.WriteString(k)
		if err != nil {
			return 0, err
		}
		t += c
		_, err = w.WriteString("=\"")
		if err != nil {
			return 0, err
		}
		t += 1
		c, err = w.WriteString(Escape(v))
		if err != nil {
			return 0, err
		}
		t += c
		_, err = w.WriteString("\"")
		if err != nil {
			return 0, err
		}
	}
	return t, nil
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
