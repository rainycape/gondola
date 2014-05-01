package facebook

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// Stats represents information returned by Facebook's
// links.getStats method. Fields are self explanatory.
type Stats struct {
	Link       string `xml:"url"`
	Normalized string `xml:"normalized_url"`
	Shares     int    `xml:"share_count"`
	Likes      int    `xml:"like_count"`
	Comments   int    `xml:"comment_count"`
	Clicks     int    `xml:"click_count"`
}

func (app *App) Stats(urls []string) (map[string]*Stats, error) {
	escaped := make([]string, len(urls))
	for ii, v := range urls {
		escaped[ii] = url.QueryEscape(v)
	}
	data := strings.Join(escaped, ",")
	endPoint := fmt.Sprintf("http://api.facebook.com/restserver.php?method=links.getStats&urls=%s", data)
	resp, err := app.client().Get(endPoint)
	if err != nil {
		return nil, err
	}
	stats := make(map[string]*Stats, len(urls))
	defer resp.Body.Close()
	dec := xml.NewDecoder(resp.Body)
	for {
		token, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "link_stat" {
				var stat *Stats
				if err := dec.DecodeElement(&stat, &t); err != nil {
					return nil, err
				}
				stats[stat.Link] = stat
			}
		}
	}
	return stats, nil
}
