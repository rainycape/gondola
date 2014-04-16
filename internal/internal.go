package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gnd.la/util/yaml"
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

func AppEngineAppId() string {
	var m map[string]interface{}
	if err := yaml.UnmarshalFile("app.yaml", &m); err == nil {
		// XXX: your-app-id is the default in app.yaml in GAE templates, found
		// in the gondolaweb repository. Keep these in sync.
		if id, ok := m["application"].(string); ok && id != "your-app-id" {
			return id
		}
	}
	return ""
}

func AppEngineAppHost() string {
	if id := AppEngineAppId(); id != "" {
		return fmt.Sprintf("http://%s.appspot.com", id)
	}
	return ""
}

func init() {
	inTest = strings.Contains(os.Args[0], string(filepath.Separator)+"_test"+string(filepath.Separator))
}
