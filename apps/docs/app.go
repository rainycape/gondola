package docs

import (
	"gnd.la/app"
	"gnd.la/app/reusableapp"
	"gnd.la/apps/docs/doc"
	_ "gnd.la/template/assets/sass" // import the scss compiler for docs.scss
)

// Group represents a group of packages to be displayed under the same
// title. Note that all subpackages of any included package will also
// be listed. Packages must be referred by their import path (e.g.
// example.com/pkg).
type Group struct {
	Title    string
	Packages []string
}

type Options struct {
	Groups []*Group
}

type appData struct {
	Groups      []*Group
	Environment *doc.Environment
}

type App struct {
	reusableapp.App
}

func (a *App) Environment() *doc.Environment {
	return a.Data().(*appData).Environment
}

func New(opts Options) *App {
	data := &appData{
		Groups: opts.Groups,
	}
	a := reusableapp.New(reusableapp.Options{
		Name:          "Docs",
		AssetsData:    assetsData,
		TemplatesData: tmplData,
		Data:          data,
	})
	a.Prefix = "/doc/"
	reverseDoc := func(s string) string { return a.MustReverse(PackageHandlerName, s) }
	reverseSource := func(s string) string { return a.MustReverse(SourceHandlerName, s) }
	data.Environment = doc.NewEnvironment(reverseDoc, reverseSource)
	data.Environment.Set(envAppKey, a)
	a.AddTemplateVars(map[string]interface{}{
		"List":    ListHandlerName,
		"StdList": StdListHandlerName,
		"Package": PackageHandlerName,
		"Source":  SourceHandlerName,
	})
	a.Handle("^/$", ListHandler, app.NamedHandler(ListHandlerName))
	a.Handle("^/pkg/std/?", StdListHandler, app.NamedHandler(StdListHandlerName))
	a.Handle("^/pkg/(.+)", PackageHandler, app.NamedHandler(PackageHandlerName))
	a.Handle("^/src/(.+)", SourceHandler, app.NamedHandler(SourceHandlerName))
	return &App{App: *a}
}

func appDocGroups(ctx *app.Context) []*Group {
	data, _ := reusableapp.Data(ctx).(*appData)
	if data != nil {
		return data.Groups
	}
	return nil
}

type envKey int

const (
	envAppKey envKey = iota
)
