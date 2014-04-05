package articles

import (
	"fmt"
	"gnd.la/app"
	"gnd.la/template"
)

func reverseArticle(ctx *app.Context, article interface{}) (string, error) {
	return reverseAppArticle(ctx.App(), article)
}

func reverseAppArticle(a *app.App, article interface{}) (string, error) {
	articles := AppArticles(a)
	switch x := article.(type) {
	case string:
		for _, v := range articles {
			if v.Id == x {
				return a.Reverse(ArticleHandlerName, v.Slug())
			}
		}
		return "", fmt.Errorf("no article with id %q found", x)
	case *Article:
		return a.Reverse(ArticleHandlerName, x.Slug())
	case Article:
		return a.Reverse(ArticleHandlerName, x.Slug())
	}
	return "", fmt.Errorf("can't reverse Article from %T, must be *Article or string (article id)", article)
}

func init() {
	template.AddFuncs(template.FuncMap{
		"reverse_article":     reverseArticle,
		"reverse_app_article": reverseAppArticle,
	})
}
