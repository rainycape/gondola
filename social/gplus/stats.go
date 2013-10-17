package gplus

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	rpc                = "https://clients6.google.com/rpc?key=AIzaSyCKSbrvQasunBoV16zDH9R33D88CeLr9gQ"
	errInvalidResponse = errors.New("invalid JSON response")
)

type LinkStats struct {
	Normalized string
	Count      int
}

func GetLinkStats(url string) (*LinkStats, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	body := fmt.Sprintf(`[{
	    "method":"pos.plusones.get",
	    "id":"p",
	    "params":{
		"nolog":true,
		"id":"%s",
		"source":"widget",
		"userId":"@viewer",
		"groupId":"@self"
	    },
	    "jsonrpc":"2.0",
	    "key":"p",
	    "apiVersion":"v1"
	}]`, url)
	resp, err := Client.Post(rpc, "application/json", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var res []map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&res); err != nil {
		return nil, err
	}
	if len(res) != 1 {
		return nil, errInvalidResponse
	}
	result, ok := res[0]["result"].(map[string]interface{})
	if !ok {
		return nil, errInvalidResponse
	}
	normalized, ok := result["id"].(string)
	if !ok {
		return nil, errInvalidResponse
	}
	metadata, ok := result["metadata"].(map[string]interface{})
	if !ok {
		return nil, errInvalidResponse
	}
	counts, ok := metadata["globalCounts"].(map[string]interface{})
	if !ok {
		return nil, errInvalidResponse
	}
	count, ok := counts["count"].(float64)
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
