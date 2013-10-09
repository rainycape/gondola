package reddit

import (
	"time"
)

type Listing struct {
	Stories []*Story
}

func (l *Listing) Before() string {
	if len(l.Stories) > 0 {
		return "t3_" + l.Stories[0].Id
	}
	return ""
}

func (l *Listing) After() string {
	if length := len(l.Stories); length > 0 {
		return "t3_" + l.Stories[length-1].Id
	}
	return ""
}

/* URL and ThumbnailURL are strings so
   they can be serialized with gob
*/

type Story struct {
	Id           string
	Title        string
	Self         bool
	URL          string
	ThumbnailURL string
	Subreddit    string
	Score        int
	NumComments  int
	Author       string
	SelfText     string
	SelfHtml     string
	Nsfw         bool
	Created      time.Time
}
