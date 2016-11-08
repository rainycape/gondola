package articles

import (
	"path"
	"strings"

	"gnd.la/app"
	"gnd.la/apps/articles/article"
	"gnd.la/util/generic"

	"github.com/rainycape/vfs"
)

const (
	articlesKey = "articles"
)

var (
	extensions = map[string]bool{
		".html": true,
		".md":   true,
		".txt":  true,
	}
)

// List returns the articles found in the given directory in
// the fs argument, recursively.
func List(fs vfs.VFS, dir string) ([]*article.Article, error) {
	files, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var articles []*article.Article
	for _, v := range files {
		p := path.Join(dir, v.Name())
		if v.IsDir() {
			dirArticles, err := List(fs, p)
			if err != nil {
				return nil, err
			}
			articles = append(articles, dirArticles...)
			continue
		}
		if !extensions[strings.ToLower(path.Ext(p))] {
			// Not a recognized extension
			continue
		}
		article, err := article.Open(fs, p)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	sortArticles(articles)
	return articles, nil
}

// Load loads articles from the given directory in the given fs
// into the given app.
func (a *ArticlesApp) Load(fs vfs.VFS, dir string) ([]*article.Article, error) {
	articles, err := List(fs, dir)
	if err != nil {
		return nil, err
	}
	a.Articles = append(a.Articles, articles...)
	sortArticles(a.Articles)
	return articles, nil
}

// LoadDir works like Load, but loads the articles from the given directory
// in the local filesystem.
func (a *ArticlesApp) LoadDir(dir string) ([]*article.Article, error) {
	fs, err := vfs.FS(dir)
	if err != nil {
		return nil, err
	}
	return a.Load(fs, "/")
}

func sortArticles(articles []*article.Article) {
	generic.SortFunc(articles, func(a1, a2 *article.Article) bool {
		if a1.Priority < a2.Priority {
			return true
		}
		if a1.Created().Sub(a2.Created()) > 0 {
			return true
		}
		return a1.Title() < a2.Title()
	})
}

func getArticles(ctx *app.Context) []*article.Article {
	if a := getArticlesApp(ctx.App()); a != nil {
		return a.Articles
	}
	return nil
}
