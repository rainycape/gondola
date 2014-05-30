package google

import (
	"encoding/json"
	"fmt"

	"gnd.la/net/httpclient"
)

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func googleError(r *httpclient.Response) error {
	data, _ := r.ReadAll()
	var e *Error
	if err := json.Unmarshal(data, &e); err == nil && e != nil && e.Message != "" {
		return e
	}
	return fmt.Errorf("invalid status code %d: %s", r.StatusCode, string(data))
}
