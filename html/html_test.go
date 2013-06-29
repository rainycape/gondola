package html

import (
	"testing"
)

const (
	t1 = `<div>You're our 1 millionth visitor <a href="http://www.google.com">click here</a> to claim your price</div>`
	t2 = `<div class="error" id="foo"></div>`
)

func testHTML(t *testing.T, n *Node, expected string) {
	s := n.String()
	t.Logf("HTML is %q", s)
	if s != expected {
		t.Errorf("Bad tree. Want %q, got %q.", expected, s)
	}
}

func TestTree(t *testing.T) {
	div := Div(Text("You're our 1 millionth visitor "))
	div.Append(A("http://www.google.com", "click here"))
	Text(" to claim your price").AppendTo(div)
	testHTML(t, div, t1)
}

func TestAttr(t *testing.T) {
	testHTML(t, Div().AddClass("error").SetAttr("id", "foo"), t2)
}
