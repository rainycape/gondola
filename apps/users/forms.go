package users

import (
	"reflect"

	"gnd.la/app"
	"gnd.la/crypto/password"
	"gnd.la/form"
	"gnd.la/i18n"
	"gnd.la/orm"
	"gnd.la/orm/driver"
)

var (
	ErrNoPassword      = i18n.NewError("this user has no password")
	ErrInvalidPassword = i18n.NewError("invalid password")
	ErrNoUser          = i18n.NewError("unknown username or email")
)

type PasswordForm struct {
	Password        password.Password `form:",min_length=6,label=Password" json:"-"`
	ConfirmPassword password.Password `form:",optional,label=Confirm Password"`
	User            reflect.Value     `form:"-"`
}

func (f *PasswordForm) ValidateConfirmPassword() error {
	if f.ConfirmPassword != f.Password {
		return i18n.Errorf("passwords don't match")
	}
	if f.User.IsValid() {
		setUserValue(f.User, "Password", f.Password)
	}
	return nil
}

type AcceptForm struct {
	Accept bool `form:",label=I accept the Terms of Service and the Privacy Policy"`
}

func (f *AcceptForm) ValidateAccept() error {
	if !f.Accept {
		return i18n.Errorf("please, accept the Terms of Service and the Privacy Policy")
	}
	return nil
}

func SignUpForm(ctx *app.Context, user reflect.Value) *form.Form {
	passwordForm := &PasswordForm{User: user}
	acceptForm := &AcceptForm{Accept: true}
	return form.New(ctx, user.Interface(), passwordForm, acceptForm)
}

type SignIn struct {
	Username string      `form:",singleline,label=Username or Email"`
	Password string      `form:",password,label=Password"`
	From     string      `form:",optional,hidden"`
	User     interface{} `form:"-"`
}

func (s *SignIn) ValidateUsername(ctx *app.Context) error {
	norm := Normalize(s.Username)
	_, userVal := newEmptyUser()
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
		return ErrNoUser
	}
	s.User = userVal
	return nil
}

func (s *SignIn) ValidatePassword(ctx *app.Context) error {
	if s.User != nil {
		pw := getUserValue(reflect.ValueOf(s.User), "Password").(password.Password)
		if !pw.IsValid() {
			return ErrNoPassword
		}
		if pw.Check(s.Password) != nil {
			return ErrInvalidPassword
		}
	}
	return nil
}
