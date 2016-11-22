package users

import (
	"reflect"
	"strings"
	"time"

	"gnd.la/app"
	"gnd.la/net/oauth2"
	"gnd.la/social/google"
)

type Google struct {
	Id          string    `form:"-" orm:",unique" json:"id"`
	Email       string    `form:"-" orm:",unique" json:"email"`
	Name        string    `form:"-" json:"name"`
	LastName    string    `form:"-" json:"last_name"`
	Image       string    `form:"-" json:"-"`
	ImageFormat string    `form:"-" json:"-"`
	ImageURL    string    `form:"-" json:"-"`
	Token       string    `form:"-" json:"-"`
	Expires     time.Time `form:"-" json:"-"`
	Refresh     string    `form:"-" json:"-"`
}

func (g *Google) accountId() interface{} {
	return g.Id
}

func (g *Google) imageURL() string {
	return g.ImageURL
}

func (g *Google) username() string {
	return strings.Replace(g.Name, " ", "", -1)
}

func (g *Google) email() string {
	return g.Email
}

func signInGoogleTokenHandler(ctx *app.Context, client *oauth2.Client, token *oauth2.Token) {
	d := data(ctx)
	googleApp := d.opts.GoogleApp.Clone(ctx)
	user, err := userFromGoogleToken(ctx, googleApp, token)
	if err != nil {
		panic(err)
	}
	ctx.MustSignIn(asGondolaUser(user))
	redirectToFrom(ctx)
}

func jsSignInGoogleHandler(ctx *app.Context) {
	code := ctx.RequireFormValue(oauth2.Code)
	redir := "postmessage" // this is the redir value used for G+ JS sign in
	d := data(ctx)
	googleApp := d.opts.GoogleApp.Clone(ctx)
	token, err := googleApp.Exchange(redir, code)
	if err != nil {
		panic(err)
	}
	user, err := userFromGoogleToken(ctx, googleApp, token)
	if err != nil {
		panic(err)
	}
	ctx.MustSignIn(asGondolaUser(user))
	writeJSONEncoded(ctx, user)
}

func userFromGoogleToken(ctx *app.Context, googleApp *google.App, token *oauth2.Token) (reflect.Value, error) {
	person, err := googleApp.Person("me", token.Key)
	if err != nil {
		return reflect.Value{}, err
	}
	email := person.Emails[0].Value
	guser := &Google{
		Id:       person.Id,
		Email:    email,
		Name:     person.Name.Given,
		LastName: person.Name.Family,
		ImageURL: strings.Replace(person.Image, "sz=50", "sz=200", -1),
		Token:    token.Key,
		Expires:  token.Expires,
		Refresh:  token.Refresh,
	}
	return userWithSocialAccount(ctx, SocialAccountTypeGoogle, guser)
}
