package reddit

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"gnd.la/net/httpclient"
)

const (
	maxChunkSize          = 25 /* Restrictions by Reddit */
	minTimeBetweenQueries = 2 * time.Second
)

var (
	ErrNoSuchStory       = errors.New("Story with the given ID not found")
	ErrInvalidDataFormat = errors.New("Invalid data format")
)

var lastRequest struct {
	sync.RWMutex
	time time.Time
}

type App struct {
	Client     *httpclient.Client
	httpClient *httpclient.Client
}

func (a *App) Clone(ctx httpclient.Context) *App {
	ac := *a
	ac.Client = ac.Client.Clone(ctx)
	return &ac
}

func (a *App) client() *httpclient.Client {
	if a.Client != nil {
		return a.Client
	}
	if a.httpClient == nil {
		a.httpClient = httpclient.New(nil)
	}
	return a.httpClient
}

func (a *App) Story(id36 string) (*Story, error) {
	stories, err := a.Stories([]string{id36})
	if err != nil {
		return nil, err
	}
	if len(stories) == 0 {
		return nil, ErrNoSuchStory
	}
	return stories[0], nil
}

func (a *App) Stories(ids36 []string) ([]*Story, error) {
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
		resp, err := a.get(url, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Close()
		listing, err := decodeListing(resp)
		if err != nil {
			return nil, err
		}
		stories = append(stories, listing.Stories...)
	}
	return stories, nil
}

func (a *App) Subreddit(name string, source string, p *Parameter, after string, before string) (*Listing, error) {
	urlStr := fmt.Sprintf("http://www.reddit.com/r/%s/%s.json", name, source)
	values := make(url.Values)
	if p != nil && p.isValid(source) {
		for k, v := range p.values {
			values.Add(k, v)
		}
	}
	if after != "" {
		values.Add("after", after)
	}
	if before != "" {
		values.Add("before", before)
	}
	resp, err := a.get(urlStr, values)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	listing, err := decodeListing(resp)
	if err != nil {
		return nil, err
	}
	return listing, nil
}

func (a *App) get(urlStr string, values url.Values) (*httpclient.Response, error) {
	now := time.Now()
	lastRequest.RLock()
	since := now.Sub(lastRequest.time)
	lastRequest.RUnlock()
	if delta := minTimeBetweenQueries - since; delta > 0 {
		time.Sleep(delta)
	}
	lastRequest.Lock()
	lastRequest.time = now
	lastRequest.Unlock()
	return a.client().GetForm(urlStr, values)
}

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

func decodeListing(resp *httpclient.Response) (*Listing, error) {
	var value map[string]interface{}
	err := resp.JSONDecode(&value)
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
