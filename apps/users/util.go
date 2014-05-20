package users

import (
	"fmt"
	"strconv"

	"gnd.la/app"
	"gnd.la/orm"
)

func FindFreeUsername(ctx *app.Context, username string) string {
	original := username
	ii := 1
	o := ctx.Orm()
	tbl := o.TypeTable(userType)
	if tbl == nil {
		panic(fmt.Errorf("user type %s is not registered with the orm - add orm.Register(&%s{}) somewhere in your app", userType, userType.Name()))
	}
	for {
		exists, err := o.Exists(tbl, orm.Eq("NormalizedUsername", Normalize(username)))
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
