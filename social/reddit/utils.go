package reddit

import (
	"net/url"
	"strings"
)

func decodeUrl(value map[string]interface{}, key string, required bool) (string, error) {
	s, ok := value[key].(string)
	if !ok && required {
		return "", ErrInvalidDataFormat
	}
	/* Replace &amp; with & and ' 0 with %20, since
	   Reddit escapes & and does not escape whitespace
	*/
	s = strings.Replace(s, "&amp;", "&", -1)
	s = strings.Replace(s, " ", "%20", -1)
	u, err := url.Parse(s)
	if err != nil {
		if !required {
			err = nil
		}
		return "", err
	}
	return u.String(), nil
}

func cleanStoryTitle(title string) string {
	title = strings.Trim(title, "\t\n ")
	title = strings.Replace(title, "&amp;", "&", -1)
	title = strings.Replace(title, "&lt;", "<", -1)
	title = strings.Replace(title, "&gt;", ">", -1)
	return title
}

func decodeInt(value map[string]interface{}, key string) int {
	v := value[key]
	switch x := v.(type) {
	case float64:
		return int(x)
	case float32:
		return int(x)
	}
	return 0
}

func decodeInt64(value map[string]interface{}, key string) int64 {
	v := value[key]
	switch x := v.(type) {
	case float64:
		return int64(x)
	case float32:
		return int64(x)
	}
	return 0
}
