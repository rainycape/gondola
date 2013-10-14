package social

import (
	"gnd.la/social/twitter"
)

type TwitterConfig struct {
	App   *twitter.App
	Token *twitter.Token
}
