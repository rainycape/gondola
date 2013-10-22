package po

import (
	"path/filepath"
	"testing"
)

const (
	numMessages = 16
)

func testPo(t *testing.T, filename string, nm int) *Po {
	t.Logf("Parsing po file %s", filename)
	po, err := ParseFile(filename)
	if err != nil {
		t.Error(err)
	} else {
		if len(po.Messages) != nm {
			t.Errorf("invalid number of messages. Want %d, got %d", nm, len(po.Messages))
		}
		t.Logf("Attributes %v, %d messages", po.Attrs, len(po.Messages))
	}
	return po
}

func TestParsePo(t *testing.T) {
	testPo(t, filepath.Join("_test_data", "test.pot"), numMessages)
	matches, err := filepath.Glob("_test_data/*.po")
	if err != nil {
		t.Error(err)
	} else {
		for _, v := range matches {
			testPo(t, v, numMessages+1)
		}
	}
}
