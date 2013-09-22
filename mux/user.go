package mux

import (
	"gnd.la/users"
)

// UserFunc is called when getting the current signed
// in user. It receives the current context and the
// user id and must return the current user (if any).
type UserFunc func(ctx *Context, id int64) users.User

// User returns the currently signed in user, or nil if there's
// no user. In order to find the user, the mux must have a
// UserFunc defined.
func (c *Context) User() users.User {
	if c.user == nil && c.mux.userFunc != nil {
		var id int64
		err := c.Cookies().GetSecure(users.COOKIE_NAME, &id)
		if err == nil {
			c.user = c.mux.userFunc(c, id)
		}
	}
	return c.user
}

// SignIn sets the cookie for signin in the given user. The default
// cookie options for the mux are used.
func (c *Context) SignIn(user users.User) error {
	return c.Cookies().SetSecure(users.COOKIE_NAME, user.Id())
}

// MustSignIn works like SignIn, but panics if there's an error.
func (c *Context) MustSignIn(user users.User) {
	if err := c.SignIn(user); err != nil {
		panic(err)
	}
}

// SignOut deletes the signed in cookie for the current user. If there's
// no current signed in user, it does nothing.
func (c *Context) SignOut() {
	c.Cookies().Delete(users.COOKIE_NAME)
}
