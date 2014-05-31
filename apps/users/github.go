package users

import (
	"reflect"
	"time"

	"gnd.la/app"
	"gnd.la/net/oauth2"
	"gnd.la/orm"
)

var (
	signInGithubHandler = delayedHandler(func() app.Handler {
		if GithubApp != nil {
			return oauth2.Handler(signInGithubTokenHandler, GithubApp.Client, GithubScopes)
		}
		return nil
	})
)

type Github struct {
	Id          int64     `form:"-" json:"id" orm:",index,unique"`
	Username    string    `form:"-" json:"username"`
	Name        string    `form:"-" json:"name"`
	Company     string    `form:"-" json:"-"`
	Location    string    `form:"-" json:"-"`
	Email       string    `form:"-" json:"email"`
	Image       string    `form:"-" json:"-"`
	ImageFormat string    `form:"-" json:"-"`
	ImageURL    string    `form:"-" json:"-"`
	Token       string    `form:"-" json:"-"`
	Expires     time.Time `form:"-" json:"-"`
}

func signInGithubTokenHandler(ctx *app.Context, client *oauth2.Client, token *oauth2.Token) {
	const callback = "__users_github_signed_in"
	var user reflect.Value
	var err error
	if token != nil {
		user, err = userFromGithubToken(ctx, token)
		if err != nil {
			panic(err)
		}
	}
	windowCallbackHandler(ctx, user, callback)
}

func userFromGithubToken(ctx *app.Context, token *oauth2.Token) (reflect.Value, error) {
	ghUser, err := GithubApp.Clone(ctx).User("", token.Key)
	if err != nil {
		return reflect.Value{}, err
	}
	gh := &Github{
		Id:       ghUser.Id,
		Username: ghUser.Login,
		Name:     ghUser.Name,
		Company:  ghUser.Company,
		Location: ghUser.Location,
		Email:    ghUser.Email,
		ImageURL: ghUser.AvatarURL,
		Token:    token.Key,
		Expires:  token.Expires,
	}
	user, userVal := newEmptyUser()
	ok, err := ctx.Orm().One(orm.Eq("Github.Id", gh.Id), userVal)
	if err != nil {
		return reflect.Value{}, err
	}
	if ok {
		if p := getUserValue(user, "Github").(*Github); p != nil {
			gh.Image, gh.ImageFormat, gh.ImageURL = mightFetchImage(ctx, gh.ImageURL, p.Image, p.ImageFormat, p.ImageURL)
		}
		setUserValue(user, "Github", gh)
	} else {
		gh.Image, gh.ImageFormat, gh.ImageURL = fetchImage(ctx, gh.ImageURL)
		if gh.Email != "" {
			// Check if we have a user with that email. In that case
			// Add this GH account to his account
			ok, err = ctx.Orm().One(orm.Eq("NormalizedEmail", Normalize(gh.Email)), userVal)
			if err != nil {
				return reflect.Value{}, err
			}
			if ok {
				setUserValue(user, "GitHub", gh)
			}
		}
		if !ok {
			username := gh.Username
			freeUsername := FindFreeUsername(ctx, username)
			user = newUser(freeUsername)
			setUserValue(user, "AutomaticUsername", true)
			setUserValue(user, "Email", gh.Email)
			setUserValue(user, "Github", gh)
		}
	}
	ctx.Orm().MustSave(user.Interface())
	return user, nil
}
