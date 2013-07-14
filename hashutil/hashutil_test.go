package hashutil

import (
	"bytes"
	"testing"
)

func TestInputs(t *testing.T) {
	text := "foobar"
	h1 := Sha1(text)
	h2 := Sha1([]byte(text))
	h3 := Sha1(bytes.NewReader([]byte(text)))
	if h1 != h2 || h2 != h3 || h1 != h3 {
		t.Error("different hash depending on input type")
	}
}

func TestInvalidInput(t *testing.T) {
	defer func() {
		err := recover()
		t.Logf("recovered error %s", err)
		if err == nil {
			t.Error("expecting a panic when hashing invalid type")
		}
	}()
	Sha1(42)
}
