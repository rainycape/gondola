package social

import (
	"bytes"
	"gnd.la/social/twitter"
)

import (
	"fmt"
)

func Share(s Service, item *Item, config interface{}) (interface{}, error) {
	if err := validateConfig(s, config); err != nil {
		return nil, err
	}
	switch s {
	case Twitter:
		var conf *TwitterConfig
		if cp, ok := config.(*TwitterConfig); ok {
			conf = cp
		} else if c, ok := config.(TwitterConfig); ok {
			conf = &c
		}
		var buf bytes.Buffer
		buf.WriteString(item.Title)
		for _, v := range item.URLs {
			buf.WriteByte(' ')
			buf.WriteString(v.String())
		}
		for _, v := range item.Images {
			buf.WriteByte(' ')
			buf.WriteString(v.String())
		}
		tweet, err := twitter.Update(buf.String(), conf.App, conf.Token, &twitter.TweetOptions{Truncate: true})
		return tweet, err
	default:
	}
	return nil, nil
}

func validateConfig(s Service, config interface{}) error {
	switch s {
	case Twitter:
		var conf *TwitterConfig
		if cp, ok := config.(*TwitterConfig); ok {
			conf = cp
		} else if c, ok := config.(TwitterConfig); ok {
			conf = &c
		}
		if conf.App == nil {
			return fmt.Errorf("twitter app can't be nil")
		}
		if conf.Token == nil {
			return fmt.Errorf("twitter token can't be nil")
		}
	case Facebook:
		var conf *FacebookConfig
		if cp, ok := config.(*FacebookConfig); ok {
			conf = cp
		} else if c, ok := config.(FacebookConfig); ok {
			conf = &c
		}
		if conf.AccessToken == "" {
			return fmt.Errorf("facebook access_token can't be empty")
		}
	default:
		return fmt.Errorf("service %s is not supported", s)
	}
	return nil
}
