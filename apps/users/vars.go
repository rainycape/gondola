package users

import (
	"gnd.la/social/facebook"
	"gnd.la/social/github"
	"gnd.la/social/google"
	"gnd.la/social/twitter"
)

var (
	FacebookApp         *facebook.App
	FacebookPermissions []string
	GoogleApp           *google.App
	GoogleScopes        = []string{google.PlusScope, google.EmailScope}
	TwitterApp          *twitter.App
	GithubApp           *github.App
	GithubScopes        = []string{github.ScopeEmail}

	// AllowUserSignIn can be used to disable non-social sign ins. If it's set to
	// false, users will only be able to sign in using the social sign in options.
	// Note that setting this variable to false also sets AllowRegistration to false.
	AllowUserSignIn = true
	// AllowRegistration can be used to disable user registration. Only existing users
	// and social accounts will be able to log in.
	AllowRegistration = true

	SocialOrder = []string{SocialTypeFacebook, SocialTypeTwitter, SocialTypeGoogle, SocialTypeGithub}
)
