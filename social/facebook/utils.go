package facebook

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"gnd.la/net/httpclient"
)

func Code(r *http.Request) string {
	return r.FormValue("code")
}

func parseFacebookTime(timeVal string) (time.Time, error) {
	replaced := strings.Replace(timeVal, "+0000", "Z", -1)
	parsed, err := time.Parse(time.RFC3339, replaced)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func responseHasError(resp *httpclient.Response) bool {
	return resp.StatusCode == http.StatusBadRequest
}

func decodeResponseError(resp *httpclient.Response) error {
	c := &ErrorContainer{}
	if err := resp.DecodeJSON(&c); err != nil {
		return err
	}
	return fmt.Errorf("error from Facebook (type %v): %v", c.Error.Type, c.Error.Message)
}

func graphURL(path string, accessToken string) string {
	proto := "http"
	if accessToken != "" {
		proto = "https"
	}
	separator := "/"
	if strings.HasPrefix(path, "/") {
		separator = ""
	}
	return fmt.Sprintf("%v://graph.facebook.com%v%v", proto, separator, path)
}
