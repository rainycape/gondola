package users

import (
	"fmt"
	"net/url"
	"reflect"
	"time"

	"gnd.la/app"
	"gnd.la/net/oauth2"
	"gnd.la/orm"
)

var (
	fbSignInHandler app.Handler
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

func signInFacebookTokenHandler(ctx *app.Context, client *oauth2.Client, token *oauth2.Token) {
	user, err := userFromFacebookToken(ctx, token)
	if err != nil {
		panic(err)
	}
	ctx.MustSignIn(asGondolaUser(user))
	redirectToFrom(ctx)
}

func signInFacebookHandler(ctx *app.Context) {
	if fbSignInHandler == nil {
		fbSignInHandler = oauth2.Handler(signInFacebookTokenHandler, FacebookApp.Client, FacebookPermissions)
	}
	fbSignInHandler(ctx)
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
	if err != nil {
		return reflect.Value{}, err
	}
	return userFromFacebookUser(ctx, user)
}

func userFromFacebookUser(ctx *app.Context, fb *Facebook) (reflect.Value, error) {
	user, userVal := newEmptyUser()
	ok, err := ctx.Orm().One(orm.Eq("Facebook.Id", fb.Id), userVal)
	if err != nil {
		return reflect.Value{}, err
	}
	if ok {
		if p := getUserValue(user, "Facebook").(*Facebook); p != nil {
			fb.Image, fb.ImageFormat, fb.ImageURL = mightFetchImage(ctx, fb.ImageURL, p.Image, p.ImageFormat, p.ImageURL)
		}
		setUserValue(user, "Facebook", fb)
	} else {
		fb.Image, fb.ImageFormat, fb.ImageURL = fetchImage(ctx, fb.ImageURL)
		// Check email
		if fb.Email != "" {
			// Check if we have a user with that email. In that case
			// Add this FB account to his account
			ok, err = ctx.Orm().One(orm.Eq("NormalizedEmail", Normalize(fb.Email)), userVal)
			if err != nil {
				return reflect.Value{}, err
			}
			if ok {
				setUserValue(user, "Facebook", fb)
			}
		}
		if !ok {
			// This is a bit racy, but we'll live with it for now
			username := fb.Username
			if username == "" {
				username = fb.FirstName
			}
			freeUsername := FindFreeUsername(ctx, username)
			user = newUser(freeUsername)
			setUserValue(user, "AutomaticUsername", true)
			setUserValue(user, "Email", fb.Email)
			setUserValue(user, "Facebook", fb)
		}
	}
	ctx.Orm().MustSave(userVal)
	return user, nil
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
