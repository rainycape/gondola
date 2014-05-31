package users

import (
	"reflect"

	"gnd.la/app"
	"gnd.la/orm"
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

func signInTwitterUserHandler(ctx *app.Context, twUser *twitter.User, token *twitter.Token) {
	const callback = "__users_twitter_signed_in"
	var user reflect.Value
	var err error
	if twUser != nil && token != nil {
		user, err = userFromTwitterUser(ctx, TwitterApp, twUser, token)
		if err != nil {
			panic(err)
		}
	}
	windowCallbackHandler(ctx, user, callback)
}

var signInTwitterHandler = twitter.AuthHandler(TwitterApp, signInTwitterUserHandler)

func userFromTwitterUser(ctx *app.Context, app *twitter.App, twuser *twitter.User, token *twitter.Token) (reflect.Value, error) {
	user, userVal := newEmptyUser()
	ok, err := ctx.Orm().One(orm.Eq("Twitter.Id", twuser.Id), userVal)
	if err != nil {
		return reflect.Value{}, err
	}
	tw := &Twitter{
		Id:       twuser.Id,
		Username: twuser.ScreenName,
		ImageURL: twuser.ImageURL,
		Token:    token.Key,
		Secret:   token.Secret,
	}
	if ok {
		// Update info
		if p := getUserValue(user, "Twitter").(*Twitter); p != nil {
			tw.Image, tw.ImageFormat, tw.ImageURL = mightFetchImage(ctx, tw.ImageURL, p.Image, p.ImageFormat, p.ImageURL)
		}
		setUserValue(user, "Twitter", tw)
	} else {
		tw.Image, tw.ImageFormat, tw.ImageURL = fetchImage(ctx, twuser.ImageURL)
		username := FindFreeUsername(ctx, twuser.ScreenName)
		user = newUser(username)
		setUserValue(user, "AutomaticUsername", true)
		setUserValue(user, "Twitter", tw)
	}
	ctx.Orm().MustSave(user.Interface())
	return user, nil
}
