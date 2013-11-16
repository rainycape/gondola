package twitter

import (
	"time"
)

type TwitterTime struct {
	time.Time
}

func (t *TwitterTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	pt, err := time.Parse(time.RubyDate, s[1:len(s)-1])
	if err != nil {
		return err
	}
	t.Time = pt
	return nil
}

type User struct {
	Id         string      `json:"id_str"`
	ScreenName string      `json:"screen_name"`
	Name       string      `json:"name"`
	Created    TwitterTime `json:"created_at"`
	Favorites  int         `json:"favourites_count"`
	Followers  int         `json:"followers_count"`
	Following  int         `json:"following_count"`
	Friends    int         `json:"friends_count"`
	ImageURL   string      `json:"profile_image_url"`
}
