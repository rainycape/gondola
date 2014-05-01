package twitter

import (
	"gnd.la/app"
)

// Handler represents a function type which receives the
// result of authenticating a Twitter user.
type Handler func(*app.Context, *User, *Token)

// AuthHandler takes a Handler a returns a app.Handler which
// can be added to a app. When users are directed to this
// handler, they're first asked to authenticate with Twitter.
// If the user accepts, Handler is called with a non-nil user
// and a non-nil token. Otherwise, Handler is called with
// both parameters set to nil.
func AuthHandler(twApp *App, handler Handler) app.Handler {
	return func(ctx *app.Context) {
		token := ctx.FormValue("oauth_token")
		verifier := ctx.FormValue("oauth_verifier")
		cloned := twApp.Clone(ctx)
		if token != "" && verifier != "" {
			at, err := cloned.Exchange(token, verifier)
			if err != nil {
				panic(err)
			}
			user, err := cloned.Verify(at)
			if err != nil {
				panic(err)
			}
			handler(ctx, user, at)
		} else if denied := ctx.FormValue("denied"); denied != "" {
			purgeToken(denied)
			handler(ctx, nil, nil)
		} else {
			callback := ctx.URL().String()
			auth, err := cloned.Authenticate(callback)
			if err != nil {
				panic(err)
			}
			ctx.Redirect(auth, false)
		}
	}
}
