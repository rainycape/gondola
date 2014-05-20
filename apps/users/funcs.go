package users

import (
	"strconv"

	"gnd.la/app"
	"gnd.la/i18n"
	"gnd.la/orm"
)

const COOKIE_NAME = app.USER_COOKIE_NAME

var (
	errNoSuchUser = i18n.NewError("no such user")
)

func Func(ctx *app.Context, id int64) app.User {
	user, _ := Get(ctx, id)
	if user != nil {
		return user
	}
	return nil
}

func Current(ctx *app.Context) app.User {
	return ctx.User()
}

func Get(ctx *app.Context, id int64) (app.User, error) {
	_, userVal := newEmptyUser()
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
