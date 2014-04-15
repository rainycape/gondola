package internal

import (
	"os"
	"path/filepath"
	"strings"
)

var (
	inTest      bool
	inAppEngine bool
)

// InTest returns true iff called when running
// from go test.
func InTest() bool {
	return inTest
}

func InAppEngine() bool {
	return inAppEngine
}

func InAppEngineDevServer() bool {
	return os.Getenv("RUN_WITH_DEVAPPSERVER") != ""
}

func init() {
	inTest = strings.Contains(os.Args[0], string(filepath.Separator)+"_test"+string(filepath.Separator))
}
