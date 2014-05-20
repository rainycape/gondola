package users

import (
	"gnd.la/social/facebook"
	"gnd.la/social/google"
	"gnd.la/social/twitter"
)

var (
	FacebookApp         *facebook.App
	FacebookPermissions []string
	GoogleApp           *google.App
	GoogleScopes        = []string{google.PlusScope, google.EmailScope}
	TwitterApp          *twitter.App
)
