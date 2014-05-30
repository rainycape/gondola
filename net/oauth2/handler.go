package oauth2

import (
	"net/url"

	"gnd.la/app"
	"gnd.la/util/stringutil"
)

const (
	stateCookieName = "state"
	redirCookieName = "redir"
)

// OAuth2TokenHandler is a handler type which receives a *Client and a
// *Token in addition to the *app.Context. An OAuth2TokenHandler must be
// wrapped via Handler before adding it to an app.
type OAuth2TokenHandler func(ctx *app.Context, client *Client, token *Token)

func cookieName(c *Client, name string) string {
	u, err := url.Parse(c.AuthorizationURL)
	if err != nil {
		panic(err)
	}
	return u.Host + "-" + name
}

// Handler returns an app.Handler from the given OAuth2TokenHandler, client and scopes.
func Handler(handler OAuth2TokenHandler, client *Client, scopes []string) app.Handler {
	return func(ctx *app.Context) {
		code := ctx.FormValue(Code)
		if code == "" {
			// First request, redirect to authorization
			state := stringutil.Random(32)
			redir := ctx.URL().String()
			auth := client.Clone(ctx).Authorization(redir, scopes, state)
			// Save parameters
			cookies := ctx.Cookies()
			cookies.Set(cookieName(client, stateCookieName), state)
			cookies.Set(cookieName(client, redirCookieName), redir)
			ctx.Redirect(auth, false)
			return
		}
		// Got a code, exchange for the token
		state := ctx.FormValue("state")
		var savedState, redir string
		stateCookie := cookieName(client, stateCookieName)
		cookies := ctx.Cookies()
		cookies.Get(stateCookie, &savedState)
		cookies.Delete(stateCookie)
		redirCookie := cookieName(client, redirCookieName)
		cookies.Get(redirCookie, &redir)
		cookies.Delete(redirCookie)
		if state != savedState {
			ctx.Forbidden("invalid state")
			return
		}
		token, err := client.Clone(ctx).Exchange(redir, code)
		if err != nil {
			panic(err)
		}
		handler(ctx, client, token)
	}
}
