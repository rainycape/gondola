package articles

import (
	"bytes"
	"html/template"
	"path/filepath"

	"gnd.la/app"
	"gnd.la/log"

	"gopkgs.com/vfs.v1"
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
	log.Debugf("loading article from dir %s", dir)
	fs, err := vfs.FS(dir)
	if err != nil {
		panic(err)
	}
	tmpl, err := app.LoadTemplate(ctx.App(), fs, nil, base)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTo(&buf, ctx, nil); err != nil {
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
