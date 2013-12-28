package assets

import (
	"fmt"
	"strings"
)

type Attributes map[string]string

func (a Attributes) String() string {
	var attrs []string
	for k, v := range map[string]string(a) {
		attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", k, strings.Replace(v, "\"", "\\\"", -1)))
	}
	return strings.Join(attrs, " ")
}
