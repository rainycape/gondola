package facebook

import (
	"errors"
	"net/url"
	"strconv"
	"time"
)

var (
	ErrMissingAccessToken = errors.New("access_token missing from response")
	ErrMissingExpires     = errors.New("expires missing from response")
)

type Token struct {
	Key     string
	Expires time.Time
}

func ParseToken(qs string) (*Token, error) {
	values, err := url.ParseQuery(qs)
	if err != nil {
		return nil, err
	}
	key := values.Get("access_token")
	if key == "" {
		return nil, ErrMissingAccessToken
	}
	expiresIn := values.Get("expires")
	if expiresIn == "" {
		expiresIn = values.Get("expires_in")
	}
	if expiresIn == "" {
		return nil, ErrMissingExpires
	}
	var duration time.Duration
	if expiresIn == "0" {
		/* 100 years (does not really expire) */
		duration = time.Hour * 24 * 365 * 100
	} else {
		seconds, err := strconv.ParseUint(expiresIn, 0, 64)
		if err != nil {
			return nil, err
		}
		duration = time.Duration(seconds) * time.Second
	}
	expires := time.Now().UTC().Add(duration)
	token := &Token{
		Key:     key,
		Expires: expires,
	}
	return token, nil
}

type Error struct {
	Type    string
	Message string
}

func (e Error) Error() string {
	return e.Message
}

type ErrorContainer struct {
	Error Error
}

type Comment struct {
	Id        string
	From      string
	FromId    string
	Message   string
	Created   time.Time
	LikeCount int
	Comments  []*Comment /* replies */
}
