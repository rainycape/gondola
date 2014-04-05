package articles

import (
	"bytes"
	"gnd.la/app"
	"gnd.la/loaders"
	"html/template"
	"path/filepath"
)

const (
	ArticleHandlerName     = "articles-article"
	ArticleListHandlerName = "articles-list"
)

var (
	ArticleHandler     = app.NamedHandler(ArticleHandlerName, articleHandler)
	ArticleListHandler = app.NamedHandler(ArticleListHandlerName, articleListHandler)
)

func articleHandler(ctx *app.Context) {
	slug := ctx.IndexValue(0)
	var article *Article
	for _, v := range AppArticles(ctx.App()) {
		if v.Slug() == slug {
			article = v
			break
		}
	}
	if article == nil {
		for _, v := range AppArticles(ctx.App()) {
			for _, t := range v.PrevTitles {
				if titleSlug(t) == slug {
					ctx.MustRedirectReverse(true, ArticleHandlerName, v.Slug())
					return
				}
			}
		}
		ctx.NotFound("article")
		return
	}
	dir, base := filepath.Split(article.Template)
	loader := loaders.FSLoader(dir)
	articleTemplate, err := app.LoadTemplate(ctx.App(), loader, nil, base)
	if err != nil {
		panic(err)
	}
	tmpl, err := app.LinkTemplate(ctx, articleTemplate)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		panic(err)
	}
	body := buf.String()
	data := map[string]interface{}{
		"Article": article,
		"Title":   article.Title,
		"Body":    template.HTML(body),
	}
	ctx.MustExecute("article.html", data)
}

func articleListHandler(ctx *app.Context) {
	data := map[string]interface{}{
		"Articles": AppArticles(ctx.App()),
		"Title":    ctx.App().Name(),
	}
	ctx.MustExecute("list.html", data)
}
