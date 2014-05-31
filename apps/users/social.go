package users

import (
	"fmt"
	"reflect"

	"gnd.la/app"
	"gnd.la/orm"
)

const (
	SocialTypeFacebook = "Facebook"
	SocialTypeTwitter  = "Twitter"
	SocialTypeGoogle   = "Google"
	SocialTypeGithub   = "Github"
)

type socialType struct {
	Name        string       // The user field name, must be one of the SocialType.. constants
	Type        reflect.Type // Go Type used to store information
	App         interface{}  // Pointer to pointer to App variable
	ClassName   string       // Class name for the social button
	HandlerName string       // The handler for signing in
	IconName    string       // The icon name
	Popup       bool         // Wheter the JS sign in uses a manual pop-up window
	PopupWidth  int
	PopupHeight int
}

func (s *socialType) IsEnabled() bool {
	val := reflect.ValueOf(s.App)
	return !val.Elem().IsNil()
}

var (
	// This maps the social user field names to their
	// types and the app variable that must be non-nil
	// to activate that type.
	socialTypes = []*socialType{
		{
			Name:        SocialTypeFacebook,
			Type:        reflect.TypeOf((*Facebook)(nil)),
			App:         &FacebookApp,
			ClassName:   "facebook",
			HandlerName: SignInFacebookHandlerName,
			IconName:    "facebook",
		},
		{
			Name:        SocialTypeTwitter,
			Type:        reflect.TypeOf((*Twitter)(nil)),
			App:         &TwitterApp,
			ClassName:   "twitter",
			HandlerName: SignInTwitterHandlerName,
			IconName:    "twitter",
			Popup:       true,
		},
		{
			Name:        SocialTypeGoogle,
			Type:        reflect.TypeOf((*Google)(nil)),
			App:         &GoogleApp,
			ClassName:   "google",
			HandlerName: SignInGoogleHandlerName,
			IconName:    "google",
		},
		{
			Name:        SocialTypeGithub,
			Type:        reflect.TypeOf((*Github)(nil)),
			App:         &GithubApp,
			ClassName:   "github",
			HandlerName: SignInGithubHandlerName,
			IconName:    "github",
			Popup:       true,
			PopupWidth:  1000,
			PopupHeight: 800,
		},
	}

	socialTypesByName = map[string]*socialType{}
)

func enabledSocialTypes() []*socialType {
	added := make(map[string]bool)
	var types []*socialType
	for _, v := range SocialOrder {
		if added[v] {
			continue
		}
		added[v] = true
		if st := socialTypesByName[v]; st != nil && st.IsEnabled() {
			types = append(types, st)
		}
	}
	return types
}

type socialAccount interface {
	accountId() interface{}
	imageURL() string
	username() string
	email() string
}

func userWithSocialAccount(ctx *app.Context, name string, acc socialAccount) (reflect.Value, error) {
	user, userVal := newEmptyUser()
	ok, err := ctx.Orm().One(orm.Eq(name+".Id", acc.accountId()), userVal)
	if err != nil {
		return reflect.Value{}, err
	}
	acVal := reflect.Indirect(reflect.ValueOf(acc))
	imageVal := acVal.FieldByName("Image")
	imageFormatVal := acVal.FieldByName("ImageFormat")
	imageURLVal := acVal.FieldByName("ImageURL")
	if ok {
		prev := getUserValue(user, name)
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
		setUserValue(user, name, acc)
	} else {
		image, imageFormat, imageURL := fetchImage(ctx, acc.imageURL())
		imageVal.Set(reflect.ValueOf(image))
		imageFormatVal.Set(reflect.ValueOf(imageFormat))
		imageURLVal.Set(reflect.ValueOf(imageURL))
		// Check email
		if email := acc.email(); email != "" {
			// Check if we have a user with that email. In that case
			// Add this social account to his account.
			ok, err = ctx.Orm().One(orm.Eq("NormalizedEmail", Normalize(email)), userVal)
			if err != nil {
				return reflect.Value{}, err
			}
			if ok {
				setUserValue(user, name, acc)
			}
		}
		if !ok {
			// This is a bit racy, but we'll live with it for now
			username := acc.username()
			freeUsername := FindFreeUsername(ctx, username)
			user = newUser(freeUsername)
			setUserValue(user, "AutomaticUsername", true)
			setUserValue(user, "Email", acc.email())
			setUserValue(user, name, acc)
		}
	}
	ctx.Orm().MustSave(user.Interface())
	return user, nil
}

func getSocial(src interface{}) (*socialType, error) {
	switch x := src.(type) {
	case string:
		st := socialTypesByName[x]
		if st == nil {
			return nil, fmt.Errorf("no social type named %s", x)
		}
		return st, nil
	case *socialType:
		return x, nil
	}
	return nil, fmt.Errorf("invalid social identifier type %T (%v)", src, src)
}

func init() {
	for _, v := range socialTypes {
		socialTypesByName[v.Name] = v
	}
}
