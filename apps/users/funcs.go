package users

import (
	"reflect"
	"strconv"
	"time"

	"gnd.la/app"
	"gnd.la/crypto/password"
	"gnd.la/i18n"
	"gnd.la/orm"
	"gnd.la/orm/driver"
)

var (
	errNoSuchUser = i18n.NewError("no such user")
)

func userFunc(ctx *app.Context, id int64) app.User {
	user, _ := Get(ctx, id)
	if user != nil {
		return user
	}
	return nil
}

// Current returns the currently authenticated user, if any.
func Current(ctx *app.Context) app.User {
	return ctx.User()
}

// Authenticate authenticates an user using her username (or email)
// and password. This is a conveniency function for checking the values
// received by a custom handler (e.g. http auth). Otherwise, it's more convenient
// to use the builtin sign in handler in this app, which also takes care of
// cookies.
func Authenticate(ctx *app.Context, usernameOrEmail string, password string) (app.User, error) {
	user, err := getByUsernameOrEmail(ctx, usernameOrEmail)
	if err != nil {
		return nil, err
	}
	if err := validateUserPassword(ctx, user, password); err != nil {
		return nil, err
	}
	return user.(app.User), nil
}

// Create creates a new user user with the given parameters, previously checking that
// the given username and email aren't already in use. Note that the user is only created,
// no implicit sign in is performed.
func Create(ctx *app.Context, username string, email string, pw string) (app.User, error) {
	if err := validateNewUsername(ctx, username); err != nil {
		return nil, err
	}
	addr, err := validateNewEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	user, userIface := newEmptyUser(ctx)
	setUserValue(user, "Username", username)
	setUserValue(user, "NormalizedUsername", Normalize(username))
	setUserValue(user, "Email", addr)
	setUserValue(user, "NormalizedEmail", Normalize(addr))
	setUserValue(user, "Password", password.New(pw))
	setUserValue(user, "Created", time.Now().UTC())
	if _, err = ctx.Orm().Insert(userIface); err != nil {
		return nil, err
	}
	return userIface.(app.User), nil
}

func getByUsernameOrEmail(ctx *app.Context, usernameOrEmail string) (interface{}, error) {
	norm := Normalize(usernameOrEmail)
	_, userVal := newEmptyUser(ctx)
	var ok bool
	o := ctx.Orm()
	q1 := orm.Eq("User.NormalizedUsername", norm)
	q2 := orm.Eq("User.NormalizedEmail", norm)
	if o.Driver().Capabilities()&driver.CAP_OR != 0 {
		ok = o.MustOne(orm.Or(q1, q2), userVal)
	} else {
		ok = o.MustOne(q1, userVal)
		if !ok {
			ok = o.MustOne(q2, userVal)
		}
	}
	if !ok {
		return nil, ErrNoUser
	}
	return userVal, nil
}

func validateUserPassword(ctx *app.Context, user interface{}, userPw string) error {
	pw := getUserValue(reflect.ValueOf(user), "Password").(password.Password)
	if !pw.IsValid() {
		return ErrNoPassword
	}
	if pw.Check(userPw) != nil {
		return ErrInvalidPassword
	}
	return nil
}

func Get(ctx *app.Context, id int64) (app.User, error) {
	_, userVal := newEmptyUser(ctx)
	key := "gnd:la:user:" + strconv.FormatInt(id, 10)
	if ctx.Cache().Get(key, userVal) == nil {
		return userVal.(app.User), nil
	}
	ok, err := ctx.Orm().One(orm.Eq("User.UserId", id), userVal)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errNoSuchUser
	}
	ctx.Cache().Set(key, userVal, 300)
	return userVal.(app.User), nil
}
