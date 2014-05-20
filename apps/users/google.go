package users

import (
	"gnd.la/app"
	"gnd.la/orm"
	"gnd.la/social/google"
	"gnd.la/util/stringutil"
	"reflect"
	"strings"
	"time"
)

const (
	gStateCookie = "g-state"
	gRedirCookie = "g-redir"
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

func signInGoogleHandler(ctx *app.Context) {
	code := ctx.FormValue("code")
	if code != "" {
		cookies := ctx.Cookies()
		xhr := ctx.IsXHR()
		if !xhr {
			state := ctx.FormValue("state")
			var initialState string
			cookies.Get(gStateCookie, &initialState)
			cookies.Delete(gStateCookie)
			if state == "" || state != initialState {
				ctx.Forbidden("invalid state")
				return
			}
		}
		var redir string
		if xhr {
			redir = "postmessage"
		} else {
			cookies.Get(gRedirCookie, &redir)
			cookies.Delete(gRedirCookie)
		}
		token, err := GoogleApp.Clone(ctx).Exchange(code, redir)
		if err != nil {
			panic(err)
		}
		user, err := userFromGoogleToken(ctx, token)
		if err != nil {
			panic(err)
		}
		ctx.MustSignIn(asGondolaUser(user))
		if xhr {
			writeJSONEncoded(ctx, user)
		} else {
			redirectToFrom(ctx)
		}
	} else {
		cookies := ctx.Cookies()
		redir := ctx.URL().String()
		state := stringutil.Random(32)
		cookies.Set(gStateCookie, state)
		cookies.Set(gRedirCookie, redir)
		auth, err := GoogleApp.Clone(ctx).Authorize(GoogleScopes, redir, state)
		if err != nil {
			panic(err)
		}
		ctx.Redirect(auth, false)
	}
}

func userFromGoogleToken(ctx *app.Context, token *google.Token) (reflect.Value, error) {
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
	ctx.Orm().MustSave(userVal)
	return user, nil
}
