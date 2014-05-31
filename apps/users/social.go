package users

import (
	"reflect"
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

func init() {
	for _, v := range socialTypes {
		socialTypesByName[v.Name] = v
	}
}
