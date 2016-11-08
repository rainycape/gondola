package docs

import (
	"gnd.la/app"
	"gnd.la/apps/docs/doc"
	"gnd.la/kvs"
	"gnd.la/template/assets"
	_ "gnd.la/template/assets/sass"
	"gnd.la/util/apputil"
)

const (
	assetsPrefix = "/assets/"
	appName      = "Docs"
)

var (
	assetsFS    = apputil.MustOpenVFS(appName, "assets", assetsData)
	templatesFS = apputil.MustOpenVFS(appName, "tmpl", tmplData)

	getDocsApp func(kvs.Storage) *DocsApp
	setDocsApp func(kvs.Storage, *DocsApp)
)

// Group represents a group of packages to be displayed under the same
// title. Note that all subpackages of any included package will also
// be listed. Packages must be referred by their import path (e.g.
// example.com/pkg).
type Group struct {
	Title    string
	Packages []string
}

type DocsApp struct {
	*apputil.ReusableApp
	Groups      []*Group
	Environment *doc.Environment
}

func New() *DocsApp {
	a := &DocsApp{
		ReusableApp: apputil.NewReusableApp(appName),
	}
	a.Prefix = "/doc/"
	reverseDoc := func(s string) string { return a.MustReverse(PackageHandlerName, s) }
	reverseSource := func(s string) string { return a.MustReverse(SourceHandlerName, s) }
	a.Environment = doc.NewEnvironment(reverseDoc, reverseSource)
	doc.SetEnvironment(a, a.Environment)
	setDocsApp(a, a)
	setDocsApp(a.Environment, a)
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
	kvs.TypeFuncs(&getDocsApp, &setDocsApp)
}
