package messages

import (
	"bytes"
	"flag"
	"gnd.la/log"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var (
	output = flag.String("o", "", "File to extract the messages to")
)

func TestExtract(t *testing.T) {
	if testing.Verbose() {
		log.SetLevel(log.LDebug)
	}
	m, err := Extract("_test_data", DefaultExtractOptions())
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
		t.Errorf("invalid messages (expected %d bytes - got %d)", len(b), len(buf.Bytes()))
	}
	if *output != "" {
		ioutil.WriteFile(*output, buf.Bytes(), 0644)
	}
}

func init() {
	flag.Parse()
}
