package google

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func googleError(r io.Reader, statusCode int) error {
	data, _ := ioutil.ReadAll(r)
	var e *Error
	if err := json.Unmarshal(data, &e); err == nil && e != nil && e.Message != "" {
		return e
	}
	return fmt.Errorf("invalid status code %d: %s", statusCode, string(data))
}
