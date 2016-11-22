package users

import (
	"fmt"
	"os"
	"text/tabwriter"

	"gnd.la/app"
	"gnd.la/commands"
	"gnd.la/crypto/password"

	"github.com/bgentry/speakeasy"
)

func registerUser(ctx *app.Context) {
	username := ctx.RequireIndexValue(0)
	userVal, _ := newEmptyUser(ctx)
	updating := false
	if ctx.Orm().MustOne(ByUsername(username), userVal.Interface()) {
		// Updating existing user
		updating = true
	} else {
		// Creating a new one
		userVal = newUser(ctx, username)
	}
	var askPassword bool
	ctx.ParseParamValue("p", &askPassword)
	if !updating || askPassword {
		password1, err := speakeasy.Ask("Password:")
		if err != nil {
			panic(err)
		}
		password2, err := speakeasy.Ask("Confirm Password:")
		if err != nil {
			panic(err)
		}
		if password1 != password2 {
			panic(fmt.Errorf("passwords don't match"))
		}
		setUserValue(userVal, "Password", password.New(password1))
	}
	var admin bool
	ctx.ParseParamValue("s", &admin)
	setUserValue(userVal, "Admin", admin)

	var email string
	ctx.ParseParamValue("e", &email)
	if email != "" {
		setUserValue(userVal, "Email", email)
	}

	ctx.Orm().MustSave(userVal.Interface())
	ctx.Logger().Infof("saved user as %+v", userVal.Interface())
}

func listUsers(ctx *app.Context) {
	userVal, ptr := newEmptyUser(ctx)
	iter := ctx.Orm().All().Iter()
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', tabwriter.Debug)
	fmt.Fprint(w, "ID\tUsername\tEmail\tAdmin?\n")
	for iter.Next(ptr) {
		val := userVal.Elem().FieldByName("User").Interface().(User)
		fmt.Fprintf(w, "%d\t%s\t%s\t%v\n", val.UserId, val.Username, val.Email, val.Admin)
	}
	if err := w.Flush(); err != nil {
		panic(err)
	}
}

func init() {
	commands.Register(registerUser, &commands.Options{
		Usage: "[-s | -p | -e email ] <username>",
		Help:  "Registers a new user",
		Flags: commands.Flags(
			commands.BoolFlag("s", false, "Create an admin - if the user already exists is made an admin"),
			commands.BoolFlag("p", false, "Update the user password, only used when updating a user"),
			commands.StringFlag("e", "", "Email for the created user"),
		),
	})
	commands.Register(listUsers, &commands.Options{
		Help: "List all registered users",
	})
}
