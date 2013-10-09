package facebook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func ParseFacebookTime(timeVal string) (time.Time, error) {
	replaced := strings.Replace(timeVal, "+0000", "Z", -1)
	parsed, err := time.Parse(time.RFC3339, replaced)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func ResponseHasError(resp *http.Response) bool {
	return resp.StatusCode == http.StatusBadRequest
}

func DecodeResponseError(resp *http.Response) error {
	c := &ErrorContainer{}
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(&c)
	if err != nil {
		return err
	}
	return fmt.Errorf("Error from Facebook (type %v): %v", c.Error.Type, c.Error.Message)
}
