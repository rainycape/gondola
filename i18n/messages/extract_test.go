package messages

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestExtract(t *testing.T) {
	m, err := Extract("_test_data", DefaultFunctions(), DefaultTypes(), DefaultTagFields())
	if err != nil {
		t.Error(err)
	}
	var buf bytes.Buffer
	if err := Write(&buf, m); err != nil {
		t.Error(err)
	}
	t.Logf("Messages:\n%s", string(buf.Bytes()))
	b, err := ioutil.ReadFile(filepath.Join("_test_data", "test.pot"))
	if err != nil {
		t.Error(err)
	}
	if len(b) != len(buf.Bytes()) || bytes.Compare(b, buf.Bytes()) != 0 {
		t.Errorf("invalid messages (%d / %d)", len(b), len(buf.Bytes()))
	}
}
