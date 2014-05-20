package users

import (
	"fmt"
	"gnd.la/app"
	"gnd.la/orm"
	"gnd.la/social/facebook"
	"gnd.la/util/stringutil"
	"reflect"
	"time"
)

const (
	fbStateCookie = "fb-state"
	fbRedirCookie = "fb-redir"
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

func signInFacebookHandler(ctx *app.Context) {
	code := ctx.FormValue("code")
	if code != "" {
		cookies := ctx.Cookies()
		state := ctx.FormValue("state")
		var savedState string
		if err := cookies.Get(fbStateCookie, &savedState); err != nil || state != savedState {
			ctx.Forbidden("invalid state")
			return
		}
		cookies.Delete(fbStateCookie)
		var redir string
		cookies.Get(fbRedirCookie, &redir)
		cookies.Delete(fbRedirCookie)
		user, err := userFromFacebookCode(ctx, code, redir)
		if err != nil {
			panic(err)
		}
		ctx.MustSignIn(asGondolaUser(user))
		redirectToFrom(ctx)
	} else {
		cookies := ctx.Cookies()
		redir := ctx.URL().String()
		state := stringutil.Random(32)
		cookies.Set(fbStateCookie, state)
		cookies.Set(fbRedirCookie, redir)
		ctx.Redirect(FacebookApp.Clone(ctx).AuthURL(redir, FacebookPermissions, state), false)
	}
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
	user, err := userFromFacebookCode(ctx, code, "")
	if err != nil {
		panic(err)
	}
	ctx.MustSignIn(asGondolaUser(user))
	writeJSONEncoded(ctx, user)
}

func fetchFacebookUser(ctx *app.Context, token *facebook.Token) (*Facebook, error) {
	fields := "id,name,first_name,last_name,email,username,picture.width(200),picture.height(200)"
	info, err := FacebookApp.Clone(ctx).Get("/me", map[string]string{"fields": fields}, token.Key)
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

func userFromFacebookCode(ctx *app.Context, code string, redir string) (reflect.Value, error) {
	token, err := FacebookApp.Clone(ctx).ExchangeCode(code, redir, true)
	if err != nil {
		return reflect.Value{}, err
	}
	user, err := fetchFacebookUser(ctx, token)
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
