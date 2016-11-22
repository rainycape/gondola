package users

import (
	"fmt"
	"strconv"

	"gnd.la/app"
	"gnd.la/i18n"
	"gnd.la/net/mail"
	"gnd.la/orm"
)

func FindFreeUsername(ctx *app.Context, username string) string {
	userType := getUserType(ctx)
	original := username
	ii := 1
	o := ctx.Orm()
	tbl := o.TypeTable(userType)
	if tbl == nil {
		panic(fmt.Errorf("user type %s is not registered with the orm - add orm.Register(&%s{}) somewhere in your app", userType, userType.Name()))
	}
	for {
		exists, err := o.Exists(tbl, orm.Eq("User.NormalizedUsername", Normalize(username)))
		if err != nil {
			panic(err)
		}
		if !exists {
			break
		}
		username = original + strconv.Itoa(ii)
		ii++
	}
	return username
}

func redirectToFrom(ctx *app.Context) {
	from := ctx.FormValue(app.SignInFromParameterName)
	if from == "" {
		from = "/"
	}
	ctx.Redirect(from, false)
}

func validateNewUsername(ctx *app.Context, username string) error {
	userType := getUserType(ctx)
	found, err := ctx.Orm().Exists(ctx.Orm().TypeTable(userType), ByUsername(username))
	if err != nil {
		return err
	}
	if found {
		return i18n.Errorf("username %q is already in use", username)
	}
	return nil
}

func validateNewEmail(ctx *app.Context, email string) (string, error) {
	addr, err := mail.Validate(email, true)
	if err != nil {
		return "", i18n.Errorf("this does not look like a valid email address")
	}
	userType := getUserType(ctx)
	found, err := ctx.Orm().Exists(ctx.Orm().TypeTable(userType), ByEmail(addr))
	if err != nil {
		return "", err
	}
	if found {
		return "", i18n.Errorf("email %q is already in use", addr)
	}
	return addr, nil
}
