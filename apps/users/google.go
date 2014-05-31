package users

import (
	"reflect"
	"strings"
	"time"

	"gnd.la/app"
	"gnd.la/net/oauth2"
	"gnd.la/orm"
)

var (
	signInGoogleHandler = delayedHandler(func() app.Handler {
		if GoogleApp != nil {
			return oauth2.Handler(signInGoogleTokenHandler, GoogleApp.Client, GoogleScopes)
		}
		return nil
	})
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

func signInGoogleTokenHandler(ctx *app.Context, client *oauth2.Client, token *oauth2.Token) {
	user, err := userFromGoogleToken(ctx, token)
	if err != nil {
		panic(err)
	}
	ctx.MustSignIn(asGondolaUser(user))
	redirectToFrom(ctx)
}

func jsSignInGoogleHandler(ctx *app.Context) {
	code := ctx.RequireFormValue(oauth2.Code)
	redir := "postmessage" // this is the redir value used for G+ JS sign in
	token, err := GoogleApp.Clone(ctx).Exchange(redir, code)
	if err != nil {
		panic(err)
	}
	user, err := userFromGoogleToken(ctx, token)
	if err != nil {
		panic(err)
	}
	ctx.MustSignIn(asGondolaUser(user))
	writeJSONEncoded(ctx, user)
}

func userFromGoogleToken(ctx *app.Context, token *oauth2.Token) (reflect.Value, error) {
	person, err := GoogleApp.Clone(ctx).Person("me", token.Key)
	if err != nil {
		return reflect.Value{}, err
	}
	user, userVal := newEmptyUser()
	ok, err := ctx.Orm().One(orm.Eq("Google.Id", person.Id), userVal)
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
	if ok {
		// Update info
		if p := getUserValue(user, "Google").(*Google); p != nil {
			guser.Image, guser.ImageFormat, guser.ImageURL = mightFetchImage(ctx, guser.ImageURL, p.Image, p.ImageFormat, p.ImageURL)
		}
		setUserValue(user, "Google", guser)
	} else {
		guser.Image, guser.ImageFormat, guser.ImageURL = fetchImage(ctx, guser.ImageURL)
		// Check if there's an already existing user with the same email
		ok, err = ctx.Orm().One(orm.Eq("NormalizedEmail", Normalize(email)), userVal)
		if err != nil {
			return reflect.Value{}, err
		}
		if ok {
			setUserValue(user, "Google", guser)
		} else {
			// Create a new account
			username := FindFreeUsername(ctx, strings.Replace(person.Name.Given, " ", "", -1))
			user = newUser(username)
			setUserValue(user, "AutomaticUsername", true)
			setUserValue(user, "Email", email)
			setUserValue(user, "Google", guser)
		}
	}
	ctx.Orm().MustSave(user.Interface())
	return user, nil
}
