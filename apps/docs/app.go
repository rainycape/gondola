package docs

import (
	"gnd.la/app"
	"gnd.la/apps/docs/doc"
	"gnd.la/template/assets"
	"gnd.la/util/apputil"
)

const (
	assetsPrefix = "/assets/"
	appName      = "Docs"
)

var (
	assetsFS    = apputil.MustOpenVFS(appName, "assets", assetsData)
	templatesFS = apputil.MustOpenVFS(appName, "tmpl", tmplData)
)

func New() *app.App {
	a := app.New()
	doc.DefaultContext.App = a
	a.SetName(appName)
	manager := assets.New(assetsFS, assetsPrefix)
	a.SetAssetsManager(manager)
	a.Handle("^"+assetsPrefix, app.HandlerFromHTTPFunc(manager.Handler()))
	a.AddTemplateVars(map[string]interface{}{
		"List":    ListHandlerName,
		"StdList": StdListHandlerName,
		"Package": PackageHandlerName,
		"Source":  SourceHandlerName,
	})
	a.SetTemplatesFS(templatesFS)
	a.Handle("^/$", ListHandler, app.NamedHandler(ListHandlerName))
	a.Handle("^/pkg/std/?", StdListHandler, app.NamedHandler(StdListHandlerName))
	a.Handle("^/pkg/(.+)", PackageHandler, app.NamedHandler(PackageHandlerName))
	a.Handle("^/src/(.+)", SourceHandler, app.NamedHandler(SourceHandlerName))
	return a
}

func init() {
	doc.DefaultContext.DocHandlerName = PackageHandlerName
	doc.DefaultContext.SourceHandlerName = SourceHandlerName
}
