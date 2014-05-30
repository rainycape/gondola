package twitter

import (
	"errors"
	"net/url"
	"strings"
)

var (
	endPoint           = "http://urls.api.twitter.com/1/urls/count.json?url="
	errInvalidResponse = errors.New("invalid JSON response")
)

type Stats struct {
	Normalized string
	Count      int
}

func (a *App) stats(u string) (*Stats, error) {
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "http://" + u
	}
	resp, err := a.client().Get(endPoint + url.QueryEscape(u))
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	var res map[string]interface{}
	if err := resp.DecodeJSON(&res); err != nil {
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
	return &Stats{
		Normalized: normalized,
		Count:      int(count),
	}, nil
}

type result struct {
	url   string
	stats *Stats
	err   error
}

func (a *App) Stats(urls []string) (map[string]*Stats, error) {
	count := len(urls)
	ch := make(chan *result, count)
	for _, v := range urls {
		go func(u string) {
			stats, err := a.stats(u)
			ch <- &result{
				url:   u,
				stats: stats,
				err:   err,
			}
		}(v)
	}
	results := make(map[string]*Stats, count)
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
