package users

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"gnd.la/app"
	"gnd.la/crypto/password"
	"gnd.la/form"
	"gnd.la/i18n"
	"gnd.la/net/mail"
	"gnd.la/orm"
	"gnd.la/util/stringutil"
)

var (
	errResetExpired = errors.New("password reset expired")
)

const (
	JSSignInHandlerName         = "users-js-sign-in"
	JSSignUpHandlerName         = "users-js-sign-up"
	JSSignInFacebookHandlerName = "users-js-sign-in-facebook"
	JSSignInGoogleHandlerName   = "users-js-sign-in-google"

	SignInFacebookHandlerName = "users-sign-in-facebook"
	SignInGoogleHandlerName   = "users-sign-in-google"
	SignInTwitterHandlerName  = "users-sign-in-twitter"
	SignInGithubHandlerName   = "users-sign-in-github"
	SignUpHandlerName         = "users-sign-up"
	SignOutHandlerName        = "users-sign-out"
	ForgotHandlerName         = "users-forgot"
	ResetHandlerName          = "users-reset"

	FacebookChannelHandlerName = "users-facebook-channel"
	ImageHandlerName           = "users-image-handler"
)

var (
	SiteName                = ""
	Salt                    = []byte("gnd.la/apps/users")
	PasswordResetExpiry     = 24 * time.Hour
	SignInHandlerName       = app.SignInHandlerName
	SignInTemplateName      = "sign-in.html"
	SignInModalTemplateName = "sign-in-modal.html"
	SignUpTemplateName      = "sign-up.html"
	ForgotTemplateName      = "forgot.html"
	ResetTemplateName       = "reset.html"

	SignInHandler           = app.NamedHandler(app.SignInHandlerName, app.Anonymous(signInHandler))
	SignInFacebookHandler   = app.NamedHandler(SignInFacebookHandlerName, app.Anonymous(signInFacebookHandler))
	SignInGoogleHandler     = app.NamedHandler(SignInGoogleHandlerName, app.Anonymous(signInGoogleHandler))
	SignInTwitterHandler    = app.NamedHandler(SignInTwitterHandlerName, app.Anonymous(signInTwitterHandler))
	SignInGithubHandler     = app.NamedHandler(SignInGithubHandlerName, app.Anonymous(signInGithubHandler))
	SignUpHandler           = app.NamedHandler(SignUpHandlerName, app.Anonymous(signUpHandler))
	SignOutHandler          = app.NamedHandler(SignOutHandlerName, app.SignOutHandler)
	ForgotHandler           = app.NamedHandler(ForgotHandlerName, app.Anonymous(forgotHandler))
	ResetHandler            = app.NamedHandler(ResetHandlerName, resetHandler)
	JSSignInHandler         = app.NamedHandler(JSSignInHandlerName, app.Anonymous(jsSignInHandler))
	JSSignInFacebookHandler = app.NamedHandler(JSSignInFacebookHandlerName, app.Anonymous(jsSignInFacebookHandler))
	JSSignInGoogleHandler   = app.NamedHandler(JSSignInGoogleHandlerName, app.Anonymous(jsSignInGoogleHandler))
	JSSignUpHandler         = app.NamedHandler(JSSignUpHandlerName, app.Anonymous(jsSignUpHandler))
	FacebookChannelHandler  = app.NamedHandler(FacebookChannelHandlerName, facebookChannelHandler)
	UserImageHandler        = app.NamedHandler(ImageHandlerName, imageHandler)
)

func signInHandler(ctx *app.Context) {
	modal := ctx.FormValue("modal") != ""
	st := enabledSocialTypes()
	if !modal && !AllowUserSignIn && len(st) == 1 {
		// Redirect to the only available social sign-in
		ctx.MustRedirectReverse(false, st[0].HandlerName)
		return
	}
	from := ctx.FormValue(app.SignInFromParameterName)
	signIn := SignIn{From: from}
	form := form.New(ctx, &signIn)
	if AllowUserSignIn && form.Submitted() && form.IsValid() {
		ctx.MustSignIn(asGondolaUser(reflect.ValueOf(signIn.User)))
		ctx.RedirectBack()
		return
	}
	user, _ := newEmptyUser()
	data := map[string]interface{}{
		"SocialTypes":       st,
		"AllowUserSignIn":   AllowUserSignIn,
		"AllowRegistration": AllowRegistration,
		"From":              from,
		"SignInForm":        form,
		"SignUpForm":        SignUpForm(ctx, user),
	}
	tmpl := SignInTemplateName
	if modal && SignInModalTemplateName != "" {
		tmpl = SignInModalTemplateName
	}
	ctx.MustExecute(tmpl, data)
}

func jsSignInHandler(ctx *app.Context) {
	if !AllowUserSignIn {
		ctx.NotFound("")
		return
	}
	signIn := SignIn{}
	form := form.New(ctx, &signIn)
	if form.Submitted() && form.IsValid() {
		user := reflect.ValueOf(signIn.User)
		ctx.MustSignIn(asGondolaUser(user))
		writeJSONEncoded(ctx, user)
		return
	}
	FormErrors(ctx, form)
}

func signUpHandler(ctx *app.Context) {
	if !allowRegistration() {
		ctx.NotFound("")
		return
	}
	from := ctx.FormValue(app.SignInFromParameterName)
	user, _ := newEmptyUser()
	form := SignUpForm(ctx, user)
	if form.Submitted() && form.IsValid() {
		saveNewUser(ctx, user)
		ctx.RedirectBack()
		return
	}
	data := map[string]interface{}{
		"From":       from,
		"SignUpForm": form,
	}
	ctx.MustExecute(SignUpTemplateName, data)
}

func jsSignUpHandler(ctx *app.Context) {
	if !allowRegistration() {
		ctx.NotFound("")
		return
	}
	user, _ := newEmptyUser()
	form := SignUpForm(ctx, user)
	if form.Submitted() && form.IsValid() {
		saveNewUser(ctx, user)
		writeJSONEncoded(ctx, user)
		return
	}
	FormErrors(ctx, form)
}

func forgotHandler(ctx *app.Context) {
	if !AllowUserSignIn {
		ctx.NotFound("")
		return
	}
	var user *User
	var isEmail bool
	var sent bool
	var fields struct {
		Username         string `form:",singleline,label=Username or Email"`
		ValidateUsername func(*app.Context) error
	}
	fields.ValidateUsername = func(c *app.Context) error {
		username := Normalize(fields.Username)
		isEmail = strings.Contains(username, "@")
		var field string
		if isEmail {
			field = "User.NormalizedEmail"
		} else {
			field = "User.NormalizedUsername"
		}
		ok := c.Orm().MustOne(orm.Eq(field, username), &user)
		if !ok {
			if isEmail {
				return i18n.Errorf("address %q does not belong to any registered user", username)
			}
			return i18n.Errorf("username %q does not belong to any registered user", username)
		}
		if user.Email == "" {
			return i18n.Errorf("username %q does not have any registered emails", username)
		}
		return nil
	}
	f := form.New(ctx, &fields)
	if f.Submitted() && f.IsValid() {
		se, err := ctx.App().EncryptSigner(Salt)
		if err != nil {
			panic(err)
		}
		values := make(url.Values)
		values.Add("u", strconv.FormatInt(user.Id(), 36))
		values.Add("t", strconv.FormatInt(time.Now().Unix(), 36))
		values.Add("n", stringutil.Random(64))
		payload := values.Encode()
		p, err := se.EncryptSign([]byte(payload))
		if err != nil {
			panic(err)
		}
		abs := ctx.URL()
		reset := fmt.Sprintf("%s://%s%s?p=%s", abs.Scheme, abs.Host, ctx.MustReverse(ResetHandlerName), p)
		data := map[string]interface{}{
			"URL": reset,
		}
		from := mail.DefaultFrom()
		if from == "" {
			from = fmt.Sprintf("no-reply@%s", abs.Host)
		}
		msg := &mail.Message{
			To:      user.Email,
			From:    from,
			Subject: fmt.Sprintf(ctx.T("Reset your %s password"), SiteName),
		}
		ctx.MustSendMail("reset_password.txt", data, msg)
		sent = true
	}
	data := map[string]interface{}{
		"ForgotForm": f,
		"IsEmail":    isEmail,
		"Sent":       sent,
		"User":       user,
	}
	ctx.MustExecute(ForgotTemplateName, data)
}

func decodeResetPayload(ctx *app.Context, payload string) (reflect.Value, error) {
	se, err := ctx.App().EncryptSigner(Salt)
	if err != nil {
		return reflect.Value{}, err
	}
	value, err := se.UnsignDecrypt(payload)
	if err != nil {
		return reflect.Value{}, err
	}
	qs, err := url.ParseQuery(string(value))
	if err != nil {
		return reflect.Value{}, err
	}
	userId, err := strconv.ParseInt(qs.Get("u"), 36, 64)
	if err != nil {
		return reflect.Value{}, err
	}
	ts, err := strconv.ParseInt(qs.Get("t"), 36, 64)
	if err != nil {
		return reflect.Value{}, err
	}
	if time.Since(time.Unix(ts, 0)) > PasswordResetExpiry {
		return reflect.Value{}, errResetExpired
	}
	user, userVal := newEmptyUser()
	ok := ctx.Orm().MustOne(orm.Eq("User.UserId", userId), userVal)
	if !ok {
		return reflect.Value{}, errNoSuchUser
	}
	return user, nil
}

func resetHandler(ctx *app.Context) {
	if !AllowUserSignIn {
		ctx.NotFound("")
		return
	}
	payload := ctx.FormValue("p")
	var valid bool
	var expired bool
	var f *form.Form
	var user reflect.Value
	var err error
	var done bool
	if payload != "" {
		user, err = decodeResetPayload(ctx, payload)
		if err == nil && user.IsValid() {
			valid = true
		} else {
			if err == errResetExpired {
				expired = true
			}
		}
	}
	if valid {
		passwordForm := &PasswordForm{User: user}
		f = form.New(ctx, passwordForm)
		if f.Submitted() && f.IsValid() {
			ctx.Orm().MustSave(user.Interface())
			ctx.MustSignIn(asGondolaUser(user))
			done = true
		}
	}
	data := map[string]interface{}{
		"Valid":        valid,
		"Expired":      expired,
		"Done":         done,
		"User":         user,
		"PasswordForm": f,
		"Payload":      payload,
	}
	ctx.MustExecute(ResetTemplateName, data)
}

func FormErrors(ctx *app.Context, frm *form.Form) {
	errors := make(map[string]string)
	for _, v := range frm.Fields() {
		if ferr := v.Err(); ferr != nil {
			errors[v.HTMLName] = ferr.Error()
		}
	}
	data := map[string]interface{}{
		"errors": errors,
	}
	ctx.WriteJSON(data)
}

func saveNewUser(ctx *app.Context, user reflect.Value) {
	setUserValue(user, "Password", password.New(string(getUserValue(user, "Password").(password.Password))))
	setUserValue(user, "Created", time.Now().UTC())
	ctx.Orm().MustInsert(user.Interface())
	ctx.MustSignIn(asGondolaUser(user))
}

func delayedHandler(f func() app.Handler) app.Handler {
	var handler app.Handler
	var mu sync.Mutex
	return func(ctx *app.Context) {
		if handler == nil {
			mu.Lock()
			defer mu.Unlock()
			if handler == nil {
				handler = f()
				if handler == nil {
					ctx.NotFound("")
					return
				}
			}
		}
		handler(ctx)
	}
}

func windowCallbackHandler(ctx *app.Context, user reflect.Value, callback string) {
	inWindow := ctx.FormValue("window") != ""
	if user.IsValid() {
		ctx.MustSignIn(asGondolaUser(user))
	}
	if inWindow {
		var payload []byte
		if user.IsValid() {
			var err error
			payload, err = JSONEncode(user.Interface())
			if err != nil {
				panic(err)
			}
		}
		ctx.MustExecute("js-callback.html", map[string]interface{}{
			"Callback": callback,
			"Payload":  payload,
		})
	} else {
		if user.IsValid() {
			redirectToFrom(ctx)
		} else {
			ctx.MustRedirectReverse(false, app.SignInHandlerName)
		}
	}
}

func allowRegistration() bool {
	return AllowUserSignIn && AllowRegistration
}
