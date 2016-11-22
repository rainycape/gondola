package users

import (
	"reflect"

	"gnd.la/app"
	"gnd.la/social/twitter"
)

type Twitter struct {
	Id          string `form:"-" sql:",unique" json:"id"`
	Username    string `form:"-" json:"username"`
	Image       string `form:"-" json:"-"`
	ImageFormat string `form:"-" json:"-"`
	ImageURL    string `form:"-" json:"-"`
	Token       string `form:"-" json:"-"`
	Secret      string `form:"-" json:"-"`
}

func (t *Twitter) accountId() interface{} {
	return t.Id
}

func (t *Twitter) imageURL() string {
	return t.ImageURL
}

func (t *Twitter) username() string {
	return t.Username
}

func (t *Twitter) email() string {
	return ""
}

func signInTwitterUserHandler(ctx *app.Context, twUser *twitter.User, token *twitter.Token) {
	const callback = "__users_twitter_signed_in"
	var user reflect.Value
	var err error
	if twUser != nil && token != nil {
		tw := &Twitter{
			Id:       twUser.Id,
			Username: twUser.ScreenName,
			ImageURL: twUser.ImageURL,
			Token:    token.Key,
			Secret:   token.Secret,
		}
		user, err = userWithSocialAccount(ctx, SocialAccountTypeTwitter, tw)
		if err != nil {
			panic(err)
		}
	}
	windowCallbackHandler(ctx, user, callback)
}
