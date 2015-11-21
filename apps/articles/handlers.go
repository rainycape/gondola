package articles

import (
	"bytes"
	"html/template"
	"path"

	"gnd.la/app"
	"gnd.la/log"

	"gnd.la/apps/articles/article"

	"github.com/rainycape/vfs"
)

const (
	ArticleHandlerName     = "articles-article"
	ArticleListHandlerName = "articles-list"
)

func ArticleHandler(ctx *app.Context) {
	slug := ctx.IndexValue(0)
	var art *article.Article
	articles := AppArticles(ctx.App())
	for _, v := range articles {
		if v.Slug() == slug {
			art = v
			break
		}
	}
	if art == nil {
		for _, v := range articles {
			for _, s := range v.AllSlugs() {
				if s == slug {
					ctx.MustRedirectReverse(true, ctx.HandlerName(), s)
					return
				}
			}
		}
		ctx.NotFound("article not found")
		return
	}
	fs := vfs.Memory()
	filename := path.Base(art.Filename)
	if filename == "" {
		filename = "article.html"
	}
	if err := vfs.WriteFile(fs, filename, art.Text, 0644); err != nil {
		panic(err)
	}
	log.Debugf("loading article %s", articleId(art))
	tmpl, err := app.LoadTemplate(ctx.App(), fs, nil, filename)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTo(&buf, ctx, nil); err != nil {
		panic(err)
	}
	body := buf.String()
	data := map[string]interface{}{
		"Article": art,
		"Title":   art.Title(),
		"Body":    template.HTML(body),
	}
	ctx.MustExecute("article.html", data)
}

func ArticleListHandler(ctx *app.Context) {
	data := map[string]interface{}{
		"Articles": AppArticles(ctx.App()),
		"Title":    ctx.App().Name(),
	}
	ctx.MustExecute("list.html", data)
}
