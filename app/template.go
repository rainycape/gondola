package app

import (
	"errors"
	"io"

	"gnd.la/app/profile"
	"gnd.la/internal/devutil/devassets"
	"gnd.la/internal/templateutil"
	"gnd.la/template"
	"gnd.la/template/assets"

	"github.com/rainycape/vfs"
	"gnd.la/internal/devutil/devserver"
)

var (
	reservedVariables     = []string{"Ctx", "App", "Apps"}
	internalAssetsManager = assets.New(devassets.AssetsFS, assetsPrefix)
	profileHook           *template.Hook
	errNoLoadedTemplate   = errors.New("this template was not loaded from App.LoadTemplate nor NewTemplate")

	templateFuncs = []*template.Func{
		{Name: "t", Fn: template_t, Traits: template.FuncTraitContext},
		{Name: "tn", Fn: template_tn, Traits: template.FuncTraitContext},
		{Name: "tc", Fn: template_tc, Traits: template.FuncTraitContext},
		{Name: "tnc", Fn: template_tnc, Traits: template.FuncTraitContext},
		{Name: "app", Fn: nop},
		{Name: templateutil.BeginTranslatableBlock, Fn: nop},
		{Name: templateutil.EndTranslatableBlock, Fn: nop},
	}
)

type TemplateProcessor func(*template.Template) (*template.Template, error)

// Template is a thin wrapper around gnd.la/template.Template, which
// simplifies execution, provides extra functions, like URL
// reversing and translations, and always passes the current *Context
// as the template Context.
//
// When executing these templates, at least the @Ctx variable is always passed
// to the template, representing the current *app.Context.
// To define additional variables, use App.AddTemplateVars.
//
// Most of the time, users should not use this type directly, but rather
// Context.Execute and Context.MustExecute.
//
// To write the result of the template to an arbitraty io.Writer rather
// than to a *Context, load the template using App.LoadTemplate and then
// use Template.ExecuteTo.
type Template struct {
	tmpl *template.Template
	app  *App
}

func (t *Template) parse(file string, vars template.VarMap) error {
	if vars != nil {
		for _, k := range reservedVariables {
			vars[k] = nil
		}
	}
	return t.tmpl.ParseVars(file, vars)
}

func (t *Template) rewriteTranslationFuncs() error {
	for _, tr := range t.tmpl.Trees() {
		if err := templateutil.ReplaceTranslatableBlocks(tr, "t"); err != nil {
			return err
		}
	}
	return nil
}

func (t *Template) prepare() error {
	if err := t.rewriteTranslationFuncs(); err != nil {
		return err
	}
	if err := t.tmpl.Compile(); err != nil {
		return err
	}
	return nil
}

// reverse is passed as a template function without context, to allow
// calling reverse from asset templates
func (t *Template) reverse(name string, args ...interface{}) (string, error) {
	return t.app.reverse(name, args)
}

// Execute executes the template, writing its result to the given
// *Context. Note that Template uses an intermediate buffer, so
// nothing will be written to the *Context in case of error.
func (t *Template) Execute(ctx *Context, data interface{}) error {
	return t.ExecuteTo(ctx, ctx, data)
}

// ExecuteTo works like Execute, but allows writing the template result
// to an arbitraty io.Writer rather than the current *Context.
func (t *Template) ExecuteTo(w io.Writer, ctx *Context, data interface{}) error {
	var tvars map[string]interface{}
	var err error
	if t.app.namespace != nil {
		tvars, err = t.app.namespace.eval(ctx)
		if err != nil {
			return err
		}
	} else {
		tvars = make(map[string]interface{})
	}
	tvars["Ctx"] = ctx
	return t.tmpl.ExecuteContext(w, data, ctx, tvars)
}

func template_t(ctx *Context, str string) string {
	return ctx.T(str)
}

func template_tn(ctx *Context, singular string, plural string, n int) string {
	return ctx.Tn(singular, plural, n)
}

func template_tc(ctx *Context, context string, str string) string {
	return ctx.Tc(context, str)
}

func template_tnc(ctx *Context, context string, singular string, plural string, n int) string {
	return ctx.Tnc(context, singular, plural, n)
}

func newTemplate(app *App, fs vfs.VFS, manager *assets.Manager) *Template {
	t := &Template{tmpl: template.New(fs, manager), app: app}
	if app.cfg != nil {
		t.tmpl.Debug = app.cfg.TemplateDebug
	}
	t.tmpl.Funcs(templateFuncs).Funcs([]*template.Func{
		{Name: "reverse", Fn: t.reverse, Traits: template.FuncTraitPure},
	})
	return t
}

func newInternalTemplate(app *App) *Template {
	return newTemplate(app, devassets.AssetsFS, internalAssetsManager)
}

// LoadTemplate loads a template for the given *App, using the given
// vfs.VFS and *assets.Manager. Note that users should rarely
// use this function and most of the time App.LoadTemplate() should
// be used. The purpose of this function is allowing apps to load
// templates from multiple sources. Note that, as opposed to App.LoadTemplate,
// this function does not perform any caching.
func LoadTemplate(app *App, fs vfs.VFS, manager *assets.Manager, name string) (*Template, error) {
	t, err := app.loadTemplate(fs, manager, name)
	if err != nil {
		return nil, err
	}
	if err := t.prepare(); err != nil {
		return nil, err
	}
	return t, nil
}

func nop() interface{} { return nil }

func init() {
	if profile.On {
		if devserver.IsActive() {
			t := newInternalTemplate(&App{})
			t.tmpl.Funcs([]*template.Func{
				{Name: "_gondola_profile_info", Fn: getProfileInfo},
				{Name: "_gondola_internal_asset", Fn: func(arg string) string {
					return internalAssetsManager.URL(arg)
				}},
			})
			if err := t.parse("profile.html", nil); err != nil {
				panic(err)
			}
			profileHook = &template.Hook{Template: t.tmpl, Position: assets.Bottom}
		}
	}
}
