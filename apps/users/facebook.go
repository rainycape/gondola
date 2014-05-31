package users

import (
	"fmt"
	"net/url"
	"reflect"
	"time"

	"gnd.la/app"
	"gnd.la/net/oauth2"
)

var (
	signInFacebookHandler = delayedHandler(func() app.Handler {
		if FacebookApp != nil {
			return oauth2.Handler(signInFacebookTokenHandler, FacebookApp.Client, FacebookPermissions)
		}
		return nil
	})
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
	resp, err := FacebookApp.Clone(ctx).ParseSignedRequest(req)
	if err != nil {
		panic(err)
	}
	// Let it crash if the data does not have the
	// specified format, this will make it easier
	// to find it if it happens.
	code := resp["code"].(string)
	token, err := FacebookApp.Clone(ctx).Exchange("", code)
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
	info, err := FacebookApp.Clone(ctx).Get("/me", values, token.Key)
	if err != nil {
		return nil, err
	}
	id := info["id"].(string)
	username, _ := info["username"].(string)
	firstName, _ := info["first_name"].(string)
	lastName, _ := info["last_name"].(string)
	name, _ := info["name"].(string)
	email, _ := info["email"].(string)
	picture := facebookUserImage(info)
	return &Facebook{
		Id:        id,
		Username:  username,
		Name:      name,
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		ImageURL:  picture,
		Token:     token.Key,
		Expires:   token.Expires.UTC(),
	}, nil
}

func userFromFacebookToken(ctx *app.Context, token *oauth2.Token) (reflect.Value, error) {
	extended, err := FacebookApp.Clone(ctx).Extend(token)
	if err != nil {
		return reflect.Value{}, err
	}
	user, err := fetchFacebookUser(ctx, extended)
	return userWithSocialAccount(ctx, SocialTypeFacebook, user)
}

func facebookUserImage(user map[string]interface{}) string {
	if picture, ok := user["picture"].(map[string]interface{}); ok {
		if data, ok := picture["data"].(map[string]interface{}); ok {
			if isSil, _ := data["is_silhouette"].(bool); !isSil {
				if url, ok := data["url"].(string); ok {
					return url
				}
			}
		}
	}
	return ""
}

func facebookChannelHandler(ctx *app.Context) {
	ctx.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(ctx, "<script src=\"//connect.facebook.net/en_US/all.js\"></script>")
}
