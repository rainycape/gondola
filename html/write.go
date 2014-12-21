package html

import (
	"io"
)

type stringWriter interface {
	WriteString(s string) (n int, err error)
}

func (n *Node) WriteTo(w io.Writer) (int, error) {
	if sw, ok := w.(stringWriter); ok {
		return n.writeToStringWriter(sw)
	}
	return n.writeTo(w)
}

func (n *Node) writeTo(w io.Writer) (int, error) {
	t := 0
	switch n.Type {
	case TypeTag:
		_, err := w.Write([]byte{'<'})
		if err != nil {
			return 0, err
		}
		t += 1
		c, err := w.Write([]byte(n.Tag))
		if err != nil {
			return 0, err
		}
		t += c
		if n.Attrs != nil {
			c, err := n.Attrs.writeTo(w)
			if err != nil {
				return 0, err
			}
			t += c
		}
		_, err = w.Write([]byte{'>'})
		if err != nil {
			return 0, err
		}
		t += 1
		if ch := n.Children; ch != nil {
			c, err := ch.writeTo(w)
			if err != nil {
				return 0, err
			}
			t += c
		}
		if !n.Open {
			_, err := w.Write([]byte{'<', '/'})
			if err != nil {
				return 0, err
			}
			t += 2
			c, err := w.Write([]byte(n.Tag))
			if err != nil {
				return 0, err
			}
			t += c
			_, err = w.Write([]byte{'>'})
			if err != nil {
				return 0, err
			}
			t += 1
		}
	case TypeText:
		c, err := w.Write([]byte(n.Content))
		if err != nil {
			return 0, err
		}
		t += c
	}
	if n.Next != nil {
		c, err := n.Next.writeTo(w)
		if err != nil {
			return 0, err
		}
		t += c
	}
	return t, nil
}

func (n *Node) writeToStringWriter(w stringWriter) (int, error) {
	t := 0
	switch n.Type {
	case TypeTag:
		_, err := w.WriteString("<")
		if err != nil {
			return 0, err
		}
		t += 1
		c, err := w.WriteString(n.Tag)
		if err != nil {
			return 0, err
		}
		t += c
		if n.Attrs != nil {
			c, err := n.Attrs.writeToStringWriter(w)
			if err != nil {
				return 0, err
			}
			t += c
		}
		_, err = w.WriteString(">")
		if err != nil {
			return 0, err
		}
		t += 1
		if ch := n.Children; ch != nil {
			c, err := ch.writeToStringWriter(w)
			if err != nil {
				return 0, err
			}
			t += c
		}
		if !n.Open {
			_, err := w.WriteString("</")
			if err != nil {
				return 0, err
			}
			t += 2
			c, err := w.WriteString(n.Tag)
			if err != nil {
				return 0, err
			}
			t += c
			_, err = w.WriteString(">")
			if err != nil {
				return 0, err
			}
			t += 1
		}
	case TypeText:
		c, err := w.WriteString(n.Content)
		if err != nil {
			return 0, err
		}
		t += c
	}
	if n.Next != nil {
		c, err := n.Next.writeToStringWriter(w)
		if err != nil {
			return 0, err
		}
		t += c
	}
	return t, nil
}
