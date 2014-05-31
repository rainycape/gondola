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

	SocialOrder = []string{SocialTypeFacebook, SocialTypeTwitter, SocialTypeGoogle, SocialTypeGithub}
)
