package reddit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	maxChunkSize          = 25 /* Restrictions by Reddit */
	minTimeBetweenQueries = 2 * time.Second
)

var (
	ErrNoSuchStory       = errors.New("Story with the given ID not found")
	ErrInvalidDataFormat = errors.New("Invalid data format")
	UserAgent            string
	lastRequest          = time.Time{}
	client               = &http.Client{}
)

func decodeStory(value interface{}) (*Story, error) {
	v, ok := value.(map[string]interface{})
	if !ok {
		return nil, ErrInvalidDataFormat
	}
	story, ok := v["data"].(map[string]interface{})
	if !ok {
		return nil, ErrInvalidDataFormat
	}
	id, ok := story["id"].(string)
	if !ok {
		return nil, ErrInvalidDataFormat
	}
	title, ok := story["title"].(string)
	if !ok {
		return nil, ErrInvalidDataFormat
	}
	title = cleanStoryTitle(title)
	self, ok := story["is_self"].(bool)
	if !ok {
		return nil, ErrInvalidDataFormat
	}
	storyURL, err := decodeUrl(story, "url", !self)
	if err != nil {
		return nil, err
	}
	thumbnailURL, err := decodeUrl(story, "thumbnail", false)
	if err != nil {
		return nil, err
	}
	subreddit, ok := story["subreddit"].(string)
	score := decodeInt(story, "score")
	numComments := decodeInt(story, "num_comments")
	author, ok := story["author"].(string)
	selfText := ""
	selfHtml := ""
	if self {
		selfText, ok = story["selftext"].(string)
		selfHtml, ok = story["selftext_html"].(string)
	}
	nsfw, ok := story["over_18"].(bool)
	if !ok {
		return nil, ErrInvalidDataFormat
	}
	createdUtc := decodeInt64(story, "created_utc")
	created := time.Unix(createdUtc, 0).UTC()
	return &Story{
		Id:           id,
		Title:        title,
		Self:         self,
		URL:          storyURL,
		ThumbnailURL: thumbnailURL,
		Subreddit:    subreddit,
		Score:        score,
		NumComments:  numComments,
		Author:       author,
		SelfText:     selfText,
		SelfHtml:     selfHtml,
		Nsfw:         nsfw,
		Created:      created,
	}, nil
}

func decodeListing(r io.Reader) (*Listing, error) {
	decoder := json.NewDecoder(r)
	var value map[string]interface{}
	err := decoder.Decode(&value)
	if err != nil {
		return nil, err
	}
	data, ok := value["data"].(map[string]interface{})
	if !ok {
		return nil, ErrInvalidDataFormat
	}
	children, ok := data["children"].([]interface{})
	if !ok {
		return nil, ErrInvalidDataFormat
	}
	var stories []*Story
	for _, v := range children {
		story, err := decodeStory(v)
		if err != nil {
			return nil, err
		}
		stories = append(stories, story)
	}
	return &Listing{
		Stories: stories,
	}, nil
}

func get(urlStr string, parameters map[string]string) (*http.Response, error) {
	if parameters != nil {
		var values []string
		for k, v := range parameters {
			values = append(values, fmt.Sprintf("%s=%s", k, v))
		}
		queryString := strings.Join(values, "&")
		urlStr = fmt.Sprintf("%s?%s", urlStr, queryString)
	}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	if UserAgent != "" {
		req.Header.Add("User-Agent", UserAgent)
	}
	t := time.Now()
	sub := t.Sub(lastRequest)
	if sub < 0 {
		sub = 0
	}
	delta := 2*time.Second - sub
	if delta > 0 {
		time.Sleep(delta)
	}
	lastRequest = t
	return client.Do(req)
}

func FetchStories(ids36 []string) ([]*Story, error) {
	var stories []*Story
	for len(ids36) > 0 {
		count := maxChunkSize
		if count > len(ids36) {
			count = len(ids36)
		}
		ids := make([]string, count)
		for ii := 0; ii < count; ii++ {
			id := fmt.Sprintf("t3_%s", ids36[ii])
			ids = append(ids, id)
		}
		ids36 = ids36[count:]
		url := fmt.Sprintf("http://www.reddit.com/by_id/%s/.json", strings.Join(ids, ","))
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		listing, err := decodeListing(resp.Body)
		if err != nil {
			return nil, err
		}
		stories = append(stories, listing.Stories...)
	}
	return stories, nil
}

func FetchStory(id36 string) (*Story, error) {
	stories, err := FetchStories([]string{id36})
	if err != nil {
		return nil, err
	}
	if len(stories) == 0 {
		return nil, ErrNoSuchStory
	}
	return stories[0], nil
}

func FetchSubreddit(name string, source string, p *Parameter, after string, before string) (*Listing, error) {
	urlStr := fmt.Sprintf("http://www.reddit.com/r/%s/%s.json", name, source)
	parameters := make(map[string]string)
	if p != nil && p.isValid(source) {
		parameters = p.values
	}
	if after != "" {
		parameters["after"] = after
	}
	if before != "" {
		parameters["before"] = before
	}
	resp, err := get(urlStr, parameters)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	listing, err := decodeListing(resp.Body)
	if err != nil {
		return nil, err
	}
	return listing, nil
}
