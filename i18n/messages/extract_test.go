package messages

import (
	"gondola/log"
	"os"
	"testing"
)

func TestExtract(t *testing.T) {
	log.SetLevel(log.LDebug)
	m, err := Extract("test_data", DefaultFunctions(), DefaultTypes(), DefaultTagFields())
	if err != nil {
		t.Error(err)
	}
	t.Logf("Messages %v", m)
	Write(os.Stdout, m)
}
