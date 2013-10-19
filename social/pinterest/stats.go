package pinterest

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/url"
	"strings"
)

var (
	endPoint           = "http://api.pinterest.com/v1/urls/count.json?callback=&url="
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
	// Pinterest will return the results wrapped for a callback. Unfortunately, it
	// seems there's no way to remove that.
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(d) < 2 {
		return nil, errInvalidResponse
	}
	d = d[1 : len(d)-1]
	var res map[string]interface{}
	if err := json.Unmarshal(d, &res); err != nil {
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
	for ii := 0; ii < len(urls); ii++ {
		res := <-ch
		if res.err != nil {
			return nil, res.err
		}
		results[res.url] = res.stats
	}
	close(ch)
	return results, nil
}
