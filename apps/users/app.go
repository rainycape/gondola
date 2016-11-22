package users

import (
	"fmt"
	"reflect"

	"gnd.la/app"
	"gnd.la/app/reusableapp"
	"gnd.la/social/facebook"
	"gnd.la/social/github"
	"gnd.la/social/google"
	"gnd.la/social/twitter"
	"gnd.la/template"
	"gnd.la/template/assets"
	_ "gnd.la/template/assets/sass" // import the sass compiler for the scss assets
	"gnd.la/util/generic"
	"gnd.la/util/structs"
)

type appData struct {
	opts               Options
	userType           reflect.Type
	socialAccountTypes []*socialAccountType
}

type App struct {
	reusableapp.App
}

func (a *App) Attach(parent *app.App) {
	parent.SetUserFunc(userFunc)
	a.App.Attach(parent)
}

func (d *appData) setUserType(user interface{}) {
	var typ reflect.Type
	if tt, ok := user.(reflect.Type); ok {
		typ = tt
	} else {
		typ = reflect.TypeOf(user)
	}
	if typ != nil {
		for typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
	}
	if typ == nil {
		panic(fmt.Errorf("User type is not set - configure it with users.SetType(&MyUserType{})"))
	}
	s, err := structs.NewStruct(typ, nil)
	if err != nil {
		panic(err)
	}
	if !s.Embeds(innerType) {
		panic(fmt.Errorf("invalid User type %s: must embed %s e.g type %s struct {\t\t%s\n\t...\n}", typ, innerType, typ.Name(), innerType))
	}
	for _, v := range d.enabledSocialAccountTypes() {
		if !s.Has(v.Name.String(), v.Type) {
			panic(&missingFieldError{typ, v.Name.String(), v.Type})
		}
	}
	d.userType = typ
}

func (d *appData) allowDirectSignIn() bool {
	return !d.opts.DisableDirectSignIn
}

func (d *appData) allowRegistration() bool {
	return !d.opts.disableRegistration()
}

func (d *appData) hasEnabledSocialSignin() bool {
	return len(d.enabledSocialAccountTypes()) > 0
}

func (d *appData) enabledSocialAccountTypes() []*socialAccountType {
	if d.socialAccountTypes == nil {
		st := make([]*socialAccountType, 0)
		if d.opts.FacebookApp != nil {
			st = append(st, socialAccountTypes[SocialAccountTypeFacebook])
		}
		if d.opts.TwitterApp != nil {
			st = append(st, socialAccountTypes[SocialAccountTypeTwitter])
		}
		if d.opts.GoogleApp != nil {
			st = append(st, socialAccountTypes[SocialAccountTypeGoogle])
		}
		if d.opts.GithubApp != nil {
			st = append(st, socialAccountTypes[SocialAccountTypeGithub])
		}
		positions := make(map[SocialAccountType]int)
		for k := range socialAccountTypes {
			positions[k] = -1
		}
		for ii, k := range d.opts.SocialAccountsOrder {
			positions[k] = ii
		}
		generic.SortFunc(st, func(s1, s2 *socialAccountType) bool {
			p1, p2 := positions[s1.Name], positions[s2.Name]
			if p1 == -1 {
				return false
			}
			if p2 == -1 {
				return true
			}
			return p1 < p2
		})
		d.socialAccountTypes = st
	}
	return d.socialAccountTypes
}

// Options specifies the options for a *UsersApp. See each field
// for further explanations.
type Options struct {
	// The name of the site which will appear in user visible messages
	// e.g. "Sign In to SiteName".
	SiteName string
	// The user type, represented by a instance of the type.
	// e.g.
	//
	//  users{Options{UserType: MyUser{}}}
	UserType            interface{}
	FacebookApp         *facebook.App
	FacebookPermissions []string
	GoogleApp           *google.App
	GoogleScopes        []string
	TwitterApp          *twitter.App
	GithubApp           *github.App
	GithubScopes        []string
	// SocialAccountsOrder can be used to set the order in which
	// the enabled social accounts appear on the screen, from top
	// to bottom. If empty, it defaults to:
	//
	//  []string{SocialTypeFacebook, SocialTypeTwitter, SocialTypeGoogle, SocialTypeGithub}
	SocialAccountsOrder []SocialAccountType
	// DisableDirectSignIn can be used to disable non-social sign ins. If it's set to
	// true, users will only be able to sign in using the social sign in options.
	// Note that setting this variable to true also disables user registration, regardless
	// of the value of the DisableRegistration field.
	DisableDirectSignIn bool
	// DisableRegistration can be used to disable user registration. Only existing users
	// and social accounts will be able to log in.
	DisableRegistration bool
}

func (o *Options) googleScopes() []string {
	if o.GoogleScopes != nil {
		return o.GoogleScopes
	}
	// XXX: Keep this in sync with Options.GoogleScopes
	return []string{google.PlusScope, google.EmailScope}
}

func (o *Options) githubScopes() []string {
	if o.GithubScopes != nil {
		return o.GithubScopes
	}
	// XXX: Keep this in sync with comment on Options.GithubScopes
	return []string{github.ScopeEmail}
}

func (o *Options) socialAccountsOrder() []SocialAccountType {
	if o.SocialAccountsOrder != nil {
		return o.SocialAccountsOrder
	}
	// XXX: Keep this in sync with comment on Options.SocialAccountsOrder
	return []SocialAccountType{
		SocialAccountTypeFacebook,
		SocialAccountTypeTwitter,
		SocialAccountTypeGoogle,
		SocialAccountTypeGithub,
	}
}

func (o *Options) disableRegistration() bool {
	return o.DisableRegistration || o.DisableDirectSignIn
}

type key int

const (
	appDataKey key = iota
)

func New(opts Options) *App {
	d := &appData{
		opts: opts,
	}
	d.setUserType(opts.UserType)
	a := &App{App: *reusableapp.New(reusableapp.Options{
		Name:          "Users",
		Data:          d,
		DataKey:       appDataKey,
		AssetsData:    assetsData,
		TemplatesData: tmplData,
	})}
	a.Prefix = "/users/"
	a.AddTemplateVars(map[string]interface{}{
		"SiteName":            opts.SiteName,
		"FacebookApp":         opts.FacebookApp,
		"GoogleApp":           opts.GoogleApp,
		"GoogleScopes":        opts.GoogleScopes,
		"TwitterApp":          opts.TwitterApp,
		"GithubApp":           opts.GithubApp,
		"JSSignIn":            JSSignInHandlerName,
		"JSSignInFacebook":    JSSignInFacebookHandlerName,
		"JSSignInGoogle":      JSSignInGoogleHandlerName,
		"JSSignUp":            JSSignUpHandlerName,
		"Forgot":              ForgotHandlerName,
		"Reset":               ResetHandlerName,
		"SignIn":              func() string { return SignInHandlerName },
		"SignInFacebook":      SignInFacebookHandlerName,
		"SignInGoogle":        SignInGoogleHandlerName,
		"SignInTwitter":       SignInTwitterHandlerName,
		"SignInGithub":        SignInGithubHandlerName,
		"SignUp":              SignUpHandlerName,
		"SignOut":             SignOutHandlerName,
		"FacebookChannel":     FacebookChannelHandlerName,
		"Current":             Current,
		"DisableDirectSignIn": opts.DisableDirectSignIn,
		"DisableRegistration": opts.disableRegistration(),
		"SocialAccountTypes":  d.enabledSocialAccountTypes(),
	})

	a.Handle("^/sign-in/$", SignInHandler, app.NamedHandler(SignInHandlerName))
	if opts.FacebookApp != nil {
		signInFacebookHandler := oauth2SignInHandler(signInFacebookTokenHandler,
			opts.FacebookApp.Client, opts.FacebookPermissions)
		a.Handle("^/sign-in/facebook/$", signInFacebookHandler, app.NamedHandler(SignInFacebookHandlerName))
		a.Handle("^/js/sign-in/facebook/$", JSSignInFacebookHandler, app.NamedHandler(JSSignInFacebookHandlerName))
		a.Handle("^/fb-channel/$", FacebookChannelHandler, app.NamedHandler(FacebookChannelHandlerName))
	}
	if opts.GoogleApp != nil {
		signInGoogleHandler := oauth2SignInHandler(signInGoogleTokenHandler,
			opts.GoogleApp.Client, opts.googleScopes())
		a.Handle("^/sign-in/google/$", signInGoogleHandler, app.NamedHandler(SignInGoogleHandlerName))
		a.Handle("^/js/sign-in/google/$", JSSignInGoogleHandler, app.NamedHandler(JSSignInGoogleHandlerName))
	}
	if opts.TwitterApp != nil {
		signInTwitterHandler := twitter.AuthHandler(opts.TwitterApp, signInTwitterUserHandler)
		a.Handle("^/sign-in/twitter/$", signInTwitterHandler, app.NamedHandler(SignInTwitterHandlerName))
	}
	if opts.GithubApp != nil {
		signInGithubHandler := oauth2SignInHandler(signInGithubTokenHandler,
			opts.GithubApp.Client, opts.githubScopes())
		a.Handle("^/sign-in/github/$", signInGithubHandler, app.NamedHandler(SignInGithubHandlerName))
	}
	a.Handle("^/sign-up/$", SignUpHandler, app.NamedHandler(SignUpHandlerName))
	a.Handle("^/sign-out/$", SignOutHandler, app.NamedHandler(SignOutHandlerName))
	a.Handle("^/forgot/$", ForgotHandler, app.NamedHandler(ForgotHandlerName))
	a.Handle("^/reset/$", ResetHandler, app.NamedHandler(ResetHandlerName))
	a.Handle("^/js/sign-in/$", JSSignInHandler, app.NamedHandler(JSSignInHandlerName))
	a.Handle("^/js/sign-up/$", JSSignUpHandler, app.NamedHandler(JSSignUpHandlerName))
	template.AddFuncs([]*template.Func{
		{Name: "user_image", Fn: Image, Traits: template.FuncTraitContext},
		{Name: "__users_get_social", Fn: getSocial},
	})
	a.MustLoadTemplatePlugin("users-plugin.html", assets.Bottom)
	a.MustLoadTemplatePlugin("social-button.html", assets.None)
	return a
}

func data(ctx *app.Context) *appData {
	d, _ := reusableapp.AppDataWithKey(ctx.App(), appDataKey).(*appData)
	return d
}
