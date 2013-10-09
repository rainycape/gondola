package facebook

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// LinkStats represents information returned by Facebook's
// links.getStats method. Fields are self explanatory.
type LinkStats struct {
	Link       string `xml:"url"`
	Normalized string `xml:"normalized_url"`
	Shares     int    `xml:"share_count"`
	Likes      int    `xml:"like_count"`
	Comments   int    `xml:"comment_count"`
	Clicks     int    `xml:"click_count"`
}

func GetLinkStats(url string) (*LinkStats, error) {
	stats, err := GetLinksStats([]string{url})
	return stats[url], err
}

func GetLinksStats(urls []string) (map[string]*LinkStats, error) {
	escaped := make([]string, len(urls))
	for ii, v := range urls {
		escaped[ii] = url.QueryEscape(v)
	}
	data := strings.Join(escaped, ",")
	endPoint := fmt.Sprintf("http://api.facebook.com/restserver.php?method=links.getStats&urls=%s", data)
	fmt.Println(endPoint)
	resp, err := http.Get(endPoint)
	if err != nil {
		return nil, err
	}
	stats := make(map[string]*LinkStats, len(urls))
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
				var stat *LinkStats
				if err := dec.DecodeElement(&stat, &t); err != nil {
					return nil, err
				}
				stats[stat.Link] = stat
			}
		}
	}
	return stats, nil
}
