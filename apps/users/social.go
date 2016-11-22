package users

import (
	"fmt"
	"reflect"

	"gnd.la/app"
	"gnd.la/net/oauth2"
	"gnd.la/orm"
)

type SocialAccountType string

func (s SocialAccountType) String() string { return string(s) }

const (
	SocialAccountTypeFacebook SocialAccountType = "Facebook"
	SocialAccountTypeTwitter  SocialAccountType = "Twitter"
	SocialAccountTypeGoogle   SocialAccountType = "Google"
	SocialAccountTypeGithub   SocialAccountType = "Github"
)

type socialAccountType struct {
	Name        SocialAccountType // The user struct field name, must be one of the SocialType.. constants
	Type        reflect.Type      // Go Type used to store information
	ClassName   string            // Class name for the social button
	HandlerName string            // The handler for signing in
	IconName    string            // The icon name
	Popup       bool              // Wheter the JS sign in uses a manual pop-up window
	PopupWidth  int
	PopupHeight int
}

var (
	// This maps the social user field names to their
	// types.
	socialAccountTypes = map[SocialAccountType]*socialAccountType{
		SocialAccountTypeFacebook: {
			Name:        SocialAccountTypeFacebook,
			Type:        reflect.TypeOf((*Facebook)(nil)),
			ClassName:   "facebook",
			HandlerName: SignInFacebookHandlerName,
			IconName:    "facebook",
		},
		SocialAccountTypeTwitter: {
			Name:        SocialAccountTypeTwitter,
			Type:        reflect.TypeOf((*Twitter)(nil)),
			ClassName:   "twitter",
			HandlerName: SignInTwitterHandlerName,
			IconName:    "twitter",
			Popup:       true,
		},
		SocialAccountTypeGoogle: {
			Name:        SocialAccountTypeGoogle,
			Type:        reflect.TypeOf((*Google)(nil)),
			ClassName:   "google",
			HandlerName: SignInGoogleHandlerName,
			IconName:    "google",
		},
		SocialAccountTypeGithub: {
			Name:        SocialAccountTypeGithub,
			Type:        reflect.TypeOf((*Github)(nil)),
			ClassName:   "github",
			HandlerName: SignInGithubHandlerName,
			IconName:    "github",
			Popup:       true,
			PopupWidth:  1000,
			PopupHeight: 800,
		},
	}
)

type socialAccount interface {
	accountId() interface{}
	imageURL() string
	username() string
	email() string
}

func userWithSocialAccount(ctx *app.Context, name SocialAccountType, acc socialAccount) (reflect.Value, error) {
	user, userVal := newEmptyUser(ctx)
	ok, err := ctx.Orm().One(orm.Eq(name.String()+".Id", acc.accountId()), userVal)
	if err != nil {
		return reflect.Value{}, err
	}
	acVal := reflect.Indirect(reflect.ValueOf(acc))
	imageVal := acVal.FieldByName("Image")
	imageFormatVal := acVal.FieldByName("ImageFormat")
	imageURLVal := acVal.FieldByName("ImageURL")
	if ok {
		prev := getUserValue(user, name.String())
		if prev != nil {
			prevVal := reflect.Indirect(reflect.ValueOf(prev))
			prevImage := prevVal.FieldByName("Image").String()
			prevImageFormat := prevVal.FieldByName("ImageFormat").String()
			prevImageURL := prevVal.FieldByName("ImageURL").String()
			image, imageFormat, imageURL := mightFetchImage(ctx, acc.imageURL(), prevImage, prevImageFormat, prevImageURL)
			imageVal.Set(reflect.ValueOf(image))
			imageFormatVal.Set(reflect.ValueOf(imageFormat))
			imageURLVal.Set(reflect.ValueOf(imageURL))
		}
		// Note: don't update main email, since it could
		// cause a conflict if the new email is already in the db.
		// already registered wi
		setUserValue(user, name.String(), acc)
	} else {
		image, imageFormat, imageURL := fetchImage(ctx, acc.imageURL())
		imageVal.Set(reflect.ValueOf(image))
		imageFormatVal.Set(reflect.ValueOf(imageFormat))
		imageURLVal.Set(reflect.ValueOf(imageURL))
		// Check email
		if email := acc.email(); email != "" {
			// Check if we have a user with that email. In that case
			// Add this social account to his account.
			ok, err = ctx.Orm().One(orm.Eq("User.NormalizedEmail", Normalize(email)), userVal)
			if err != nil {
				return reflect.Value{}, err
			}
			if ok {
				setUserValue(user, name.String(), acc)
			}
		}
		if !ok {
			// This is a bit racy, but we'll live with it for now
			username := acc.username()
			freeUsername := FindFreeUsername(ctx, username)
			user = newUser(ctx, freeUsername)
			setUserValue(user, "AutomaticUsername", true)
			setUserValue(user, "Email", acc.email())
			setUserValue(user, name.String(), acc)
		}
	}
	ctx.Orm().MustSave(user.Interface())
	return user, nil
}

func getSocial(src interface{}) (*socialAccountType, error) {
	switch x := src.(type) {
	case string:
		st := socialAccountTypes[SocialAccountType(x)]
		if st == nil {
			return nil, fmt.Errorf("no social type named %s", x)
		}
		return st, nil
	case *socialAccountType:
		return x, nil
	}
	return nil, fmt.Errorf("invalid social identifier type %T (%v)", src, src)
}

func oauth2SignInHandler(handler oauth2.OAuth2TokenHandler, client *oauth2.Client, scopes []string) app.Handler {
	if handler != nil && client != nil {
		h := oauth2.Handler(handler, client, scopes)
		return app.Anonymous(h)
	}
	return nil
}
