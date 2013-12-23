package social

import (
	"net/url"
)

type Item struct {
	Title       string
	Description string
	Links       []*url.URL
	Images      []*url.URL
	Data        interface{}
}
