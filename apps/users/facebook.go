package users

import (
	"fmt"
	"net/url"
	"reflect"
	"time"

	"gnd.la/app"
	"gnd.la/net/oauth2"
	"gnd.la/social/facebook"
)

type Facebook struct {
	Id          string    `form:"-" sql:",unique" json:"id"`
	Username    string    `form:"-" json:"username"`
	Name        string    `form:"-" json:"name"`
	FirstName   string    `form:"-" json:"first_name"`
	LastName    string    `form:"-" json:"last_name"`
	Email       string    `form:"-" json:"email"`
	Image       string    `form:"-" json:"-"`
	ImageFormat string    `form:"-" json:"-"`
	ImageURL    string    `form:"-" json:"-"`
	Token       string    `form:"-" json:"-"`
	Expires     time.Time `form:"-" json:"-"`
}

func (f *Facebook) accountId() interface{} {
	return f.Id
}

func (f *Facebook) imageURL() string {
	return f.ImageURL
}

func (f *Facebook) username() string {
	if f.Username != "" {
		return f.Username
	}
	return f.FirstName
}

func (f *Facebook) email() string {
	return f.Email
}

func signInFacebookTokenHandler(ctx *app.Context, client *oauth2.Client, token *oauth2.Token) {
	user, err := userFromFacebookToken(ctx, token)
	if err != nil {
		panic(err)
	}
	ctx.MustSignIn(asGondolaUser(user))
	redirectToFrom(ctx)
}

func jsSignInFacebookHandler(ctx *app.Context) {
	req := ctx.FormValue("req")
	fbApp := data(ctx).opts.FacebookApp.Clone(ctx)
	resp, err := fbApp.ParseSignedRequest(req)
	if err != nil {
		panic(err)
	}
	// Let it crash if the data does not have the
	// specified format, this will make it easier
	// to find it if it happens.
	code := resp["code"].(string)
	token, err := fbApp.Exchange("", code)
	user, err := userFromFacebookToken(ctx, token)
	if err != nil {
		panic(err)
	}
	ctx.MustSignIn(asGondolaUser(user))
	writeJSONEncoded(ctx, user)
}

func fetchFacebookUser(ctx *app.Context, token *oauth2.Token) (*Facebook, error) {
	fields := "id,name,first_name,last_name,email,username,picture.width(200),picture.height(200)"
	values := make(url.Values)
	values.Set("fields", fields)
	fbApp := data(ctx).opts.FacebookApp.Clone(ctx)
	var person *facebook.Person
	if err := fbApp.Get("/me", values, token.Key, &person); err != nil {
		return nil, err
	}
	var imageURL string
	if person.Picture != nil && person.Picture.Data != nil && !person.Picture.Data.IsSilhouette {
		imageURL = person.Picture.Data.URL
	}
	return &Facebook{
		Id:        person.Id,
		Username:  person.Username,
		Name:      person.Name,
		FirstName: person.FirstName,
		LastName:  person.LastName,
		Email:     person.Email,
		ImageURL:  imageURL,
		Token:     token.Key,
		Expires:   token.Expires.UTC(),
	}, nil
}

func userFromFacebookToken(ctx *app.Context, token *oauth2.Token) (reflect.Value, error) {
	fbApp := data(ctx).opts.FacebookApp.Clone(ctx)
	extended, err := fbApp.Clone(ctx).Extend(token)
	if err != nil {
		return reflect.Value{}, err
	}
	user, err := fetchFacebookUser(ctx, extended)
	return userWithSocialAccount(ctx, SocialAccountTypeFacebook, user)
}

func FacebookChannelHandler(ctx *app.Context) {
	ctx.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(ctx, "<script src=\"//connect.facebook.net/en_US/all.js\"></script>")
}
