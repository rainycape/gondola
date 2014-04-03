package internal

import (
	"os"
	"path/filepath"
	"strings"
)

var (
	inTest bool
)

// InTest returns true iff called when running
// from go test.
func InTest() bool {
	return inTest
}

func init() {
	inTest = strings.Contains(os.Args[0], string(filepath.Separator)+"_test"+string(filepath.Separator))
}
