package twitter

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

var (
	endPoint           = "http://urls.api.twitter.com/1/urls/count.json?url="
	errInvalidResponse = errors.New("invalid JSON response")
)

type LinkStats struct {
	Normalized string
	Count      int
}

func GetLinkStats(u string) (*LinkStats, error) {
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "http://" + u
	}
	resp, err := Client.Get(endPoint + url.QueryEscape(u))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&res); err != nil {
		return nil, err
	}
	normalized, ok := res["url"].(string)
	if !ok {
		return nil, errInvalidResponse
	}
	count, ok := res["count"].(float64)
	if !ok {
		return nil, errInvalidResponse
	}
	return &LinkStats{
		Normalized: normalized,
		Count:      int(count),
	}, nil
}

type result struct {
	url   string
	stats *LinkStats
	err   error
}

func GetLinksStats(urls []string) (map[string]*LinkStats, error) {
	count := len(urls)
	ch := make(chan *result, count)
	for _, v := range urls {
		go func(u string) {
			stats, err := GetLinkStats(u)
			ch <- &result{
				url:   u,
				stats: stats,
				err:   err,
			}
		}(v)
	}
	results := make(map[string]*LinkStats, count)
	var err error
	for ii := 0; ii < len(urls); ii++ {
		res := <-ch
		if res.err != nil {
			if err == nil {
				err = res.err
			}
		} else {
			results[res.url] = res.stats
		}
	}
	close(ch)
	return results, err
}
