package articles

import (
	"gnd.la/app"
	"gnd.la/apps/articles/article"
	"gnd.la/kvs"
	"gnd.la/util/apputil"
)

const (
	appName = "Articles"
)

var (
	templatesFS = apputil.MustOpenVFS(appName, "tmpl", tmplData)

	getArticlesApp func(kvs.Storage) *ArticlesApp
	setArticlesApp func(kvs.Storage, *ArticlesApp)
)

type ArticlesApp struct {
	*apputil.ReusableApp
	Articles []*article.Article
}

func New() *ArticlesApp {
	a := &ArticlesApp{
		ReusableApp: apputil.NewReusableApp(appName),
	}
	a.AddTemplateVars(map[string]interface{}{
		"Article": ArticleHandlerName,
		"List":    ArticleListHandlerName,
	})
	a.Handle("^/(.+)/$", ArticleHandler, app.NamedHandler(ArticleHandlerName))
	a.Handle("^/$", ArticleListHandler, app.NamedHandler(ArticleListHandlerName))
	a.SetTemplatesFS(templatesFS)
	setArticlesApp(a, a)
	return a
}

func init() {
	kvs.TypeFuncs(&getArticlesApp, &setArticlesApp)
}
