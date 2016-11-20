package articles

import (
	"fmt"
	"path"

	"gnd.la/app"
	"gnd.la/app/reusableapp"
	"gnd.la/apps/articles/article"
	"gnd.la/template"
)

func articleId(art *article.Article) string {
	if art.Id != "" {
		return art.Id
	}
	if art.Filename != "" {
		base := path.Base(art.Filename)
		ext := path.Ext(base)
		return base[:len(base)-len(ext)]
	}
	return ""
}

func reverseArticle(ctx *app.Context, article interface{}) (string, error) {
	return reverseAppArticle(ctx.App(), article)
}

func reverseAppArticle(a *app.App, article interface{}) (string, error) {
	checked := make(map[*app.App]bool)
	return reverseAppsArticle(a, article, checked)
}

func reverseAppsArticle(a *app.App, art interface{}, checked map[*app.App]bool) (string, error) {
	checked[a] = true
	var articles []*article.Article
	aa, _ := reusableapp.AppData(a).(*appData)
	if aa != nil {
		articles = aa.Articles
	}
	switch x := art.(type) {
	case string:
		for _, v := range articles {
			if articleId(v) == x {
				return a.Reverse(ArticleHandlerName, v.Slug())
			}
		}
	case *article.Article:
		return a.Reverse(ArticleHandlerName, x.Slug())
	case article.Article:
		return a.Reverse(ArticleHandlerName, x.Slug())
	}
	if p := a.Parent(); p != nil && !checked[p] {
		return reverseAppsArticle(p, art, checked)
	}
	for _, v := range a.Included() {
		if checked[v] {
			continue
		}
		if url, err := reverseAppsArticle(v, art, checked); err == nil {
			return url, nil
		}
	}
	if id, ok := art.(string); ok {
		return "", fmt.Errorf("no article with id %q found", id)
	}
	return "", fmt.Errorf("can't reverse Article from %T, must be *Article or string (article id)", art)
}

func init() {
	template.AddFuncs([]*template.Func{
		{Name: "reverse_article", Fn: reverseArticle, Traits: template.FuncTraitContext},
		{Name: "reverse_app_article", Fn: reverseAppArticle},
	})
}
