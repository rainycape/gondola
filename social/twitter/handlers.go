package twitter

import (
	"gnd.la/mux"
)

// Handler represents a function type which receives the
// result of authenticating a Twitter user.
type Handler func(*mux.Context, *User, *Token)

// AuthHandler takes a Handler a returns a mux.Handler which
// can be added to a mux. When users are directed to this
// handler, they're first asked to authenticate with Twitter.
// If the user accepts, Handler is called with a non-nil user
// and a non-nil token. Otherwise, Handler is called with
// both parameters set to nil.
func AuthHandler(app *App, handler Handler) mux.Handler {
	return func(ctx *mux.Context) {
		token := ctx.FormValue("oauth_token")
		verifier := ctx.FormValue("oauth_verifier")
		if token != "" && verifier != "" {
			at, err := app.Exchange(token, verifier)
			if err != nil {
				panic(err)
			}
			user, err := app.Verify(at)
			if err != nil {
				panic(err)
			}
			handler(ctx, user, at)
		} else if denied := ctx.FormValue("denied"); denied != "" {
			purgeToken(denied)
			handler(ctx, nil, nil)
		} else {
			callback := ctx.URL().String()
			auth, err := app.Authenticate(callback)
			if err != nil {
				panic(err)
			}
			ctx.Redirect(auth, false)
		}
	}
}
