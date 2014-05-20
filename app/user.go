package app

import (
	"errors"
)

const (
	// The name of the cookie used to store the user id.
	// The cookie is signed using the gnd.la/app.App secret.
	USER_COOKIE_NAME = "user"
)

var (
	errNoUserFunc = errors.New("no UserFunc set in this App - use App.SetUserFunc() to configure one")
)

// User is the interface implemented by any struct
// that can be used to represent a user in a Gondola
// app.
type User interface {
	// Returns the numeric id of the user
	Id() int64
}

// UserFunc is called when getting the current signed
// in user. It receives the current context and the
// user id and must return the current user (if any).
type UserFunc func(ctx *Context, id int64) User

// User returns the currently signed in user, or nil if there's
// no user. In order to find the user, the App must have a
// UserFunc defined.
func (c *Context) User() User {
	if c.user == nil && c.app.userFunc != nil {
		var id int64
		err := c.Cookies().GetSecure(USER_COOKIE_NAME, &id)
		if err == nil {
			c.user = c.app.userFunc(c, id)
		}
	}
	return c.user
}

// SignIn sets the cookie for signin in the given user. The default
// cookie options for the App are used.
func (c *Context) SignIn(user User) error {
	if c.app.userFunc == nil {
		return errNoUserFunc
	}
	err := c.Cookies().SetSecure(USER_COOKIE_NAME, user.Id())
	if err != nil {
		return err
	}
	c.user = user
	return nil
}

// MustSignIn works like SignIn, but panics if there's an error.
func (c *Context) MustSignIn(user User) {
	if err := c.SignIn(user); err != nil {
		panic(err)
	}
}

// SignOut deletes the signed in cookie for the current user. If there's
// no current signed in user, it does nothing.
func (c *Context) SignOut() {
	c.Cookies().Delete(USER_COOKIE_NAME)
}
