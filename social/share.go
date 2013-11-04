package social

import (
	"bytes"
	"gnd.la/social/pinterest"
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
		conf := config.(*TwitterConfig)
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
	case Pinterest:
		conf := config.(*PinterestConfig)
		sess, err := pinterest.SignIn(conf.Account)
		if err != nil {
			return nil, err
		}
		var board *pinterest.Board
		var boardName string
		switch b := conf.Board.(type) {
		case *pinterest.Board:
			board = b
		case string:
			boardName = b
		}
		if board == nil {
			boards, err := pinterest.Boards(sess)
			if err != nil {
				return nil, err
			}
			if len(boards) > 0 {
				if boardName != "" {
					for _, v := range boards {
						if v.Name == boardName {
							board = v
							break
						}
					}
				} else {
					board = boards[0]
				}
			}
		}
		if board == nil {
			return nil, fmt.Errorf("can't find Pinterest board")
		}
		pin := &pinterest.Pin{
			Link:        item.URLs[0].String(),
			Image:       item.Images[0].String(),
			Description: item.Title,
		}
		return pinterest.Post(sess, board, pin)
	}
	return nil, fmt.Errorf("Share() does not support service %s", s)
}

func validateConfig(s Service, config interface{}) error {
	switch s {
	case Twitter:
		conf, ok := config.(*TwitterConfig)
		if !ok {
			return fmt.Errorf("%s config must be *TwitterConfig, it's %T", s, config)
		}
		if conf.App == nil {
			return fmt.Errorf("twitter app can't be nil")
		}
		if conf.Token == nil {
			return fmt.Errorf("twitter token can't be nil")
		}
	case Facebook:
		conf, ok := config.(*FacebookConfig)
		if !ok {
			return fmt.Errorf("%s config must be *FacebookConfig, it's %T", s, config)
		}
		if conf.AccessToken == "" {
			return fmt.Errorf("facebook access_token can't be empty")
		}
	case Pinterest:
		conf, ok := config.(*PinterestConfig)
		if !ok {
			return fmt.Errorf("%s config must be *PinterestConfig, it's %T", s, config)
		}
		if conf.Account == nil {
			return fmt.Errorf("pinterest account can't be empty")
		}
		if conf.Account.Username == "" {
			return fmt.Errorf("pinterest username can't be empty")
		}
		if conf.Account.Password == "" {
			return fmt.Errorf("pinterest password can't be empty")
		}
	default:
		return fmt.Errorf("service %s is not supported", s)
	}
	return nil
}
