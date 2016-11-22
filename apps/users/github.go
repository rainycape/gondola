package users

import (
	"reflect"
	"time"

	"gnd.la/app"
	"gnd.la/net/oauth2"
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

func (g *Github) accountId() interface{} {
	return g.Id
}

func (g *Github) imageURL() string {
	return g.ImageURL
}

func (g *Github) username() string {
	return g.Username
}

func (g *Github) email() string {
	return g.Email
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
	d := data(ctx)
	client := d.opts.GithubApp.Clone(ctx)
	ghUser, err := client.User("", token.Key)
	if err != nil {
		return reflect.Value{}, err
	}
	gh := &Github{
		Id:       ghUser.Id,
		Username: ghUser.Login,
		Name:     ghUser.Name,
		Company:  ghUser.Company,
		Location: ghUser.Location,
		Email:    ghUser.Email, // this is the public email, if any
		ImageURL: ghUser.AvatarURL,
		Token:    token.Key,
		Expires:  token.Expires,
	}
	// Check private emails
	emails, _ := client.Emails(token.Key)
	for _, v := range emails {
		if v.Primary {
			gh.Email = v.Address
			break
		}
	}
	if gh.Email == "" && len(emails) > 0 {
		gh.Email = emails[0].Address
	}
	return userWithSocialAccount(ctx, SocialAccountTypeGithub, gh)
}
