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
	checked := make(map[*app.App]bool)
	return reverseAppsArticle(a, article, checked)
}

func reverseAppsArticle(a *app.App, article interface{}, checked map[*app.App]bool) (string, error) {
	checked[a] = true
	articles := AppArticles(a)
	switch x := article.(type) {
	case string:
		for _, v := range articles {
			if v.Id == x {
				return a.Reverse(ArticleHandlerName, v.Slug())
			}
		}
	case *Article:
		return a.Reverse(ArticleHandlerName, x.Slug())
	case Article:
		return a.Reverse(ArticleHandlerName, x.Slug())
	}
	if p := a.Parent(); p != nil && !checked[p] {
		return reverseAppsArticle(p, article, checked)
	}
	for _, v := range a.Included() {
		if checked[v] {
			continue
		}
		if url, err := reverseAppsArticle(v, article, checked); err == nil {
			return url, nil
		}
	}
	if id, ok := article.(string); ok {
		return "", fmt.Errorf("no article with id %q found", id)
	}
	return "", fmt.Errorf("can't reverse Article from %T, must be *Article or string (article id)", article)
}

func init() {
	template.AddFuncs(template.FuncMap{
		"!reverse_article":    reverseArticle,
		"reverse_app_article": reverseAppArticle,
	})
}
