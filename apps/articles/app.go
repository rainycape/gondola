package articles

import (
	"gnd.la/app"
	"gnd.la/app/reusableapp"
	"gnd.la/apps/articles/article"
)

type appData struct {
	Articles []*article.Article
}

func articlesData(ctx *app.Context) *appData {
	d, _ := reusableapp.Data(ctx).(*appData)
	return d
}

type App struct {
	reusableapp.App
}

func New() *App {
	a := reusableapp.New(reusableapp.Options{
		Name:          "Articles",
		Data:          &appData{},
		TemplatesData: tmplData,
	})
	a.AddTemplateVars(map[string]interface{}{
		"Article": ArticleHandlerName,
		"List":    ArticleListHandlerName,
	})
	a.Handle("^/(.+)/$", ArticleHandler, app.NamedHandler(ArticleHandlerName))
	a.Handle("^/$", ArticleListHandler, app.NamedHandler(ArticleListHandlerName))
	return &App{App: *a}
}
