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

// AppArticles returns the articles loaded into the given App, or
// nil if no articles were loaded into it.
func AppArticles(a *app.App) []*article.Article {
	articles, _ := a.Get(articlesKey).([]*article.Article)
	return articles
}

func setAppArticles(a *app.App, articles []*article.Article) {
	a.Set(articlesKey, articles)
}

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
	generic.SortFunc(articles, func(a1, a2 *article.Article) bool {
		if a1.Priority < a2.Priority {
			return true
		}
		if a1.Created().Sub(a2.Created()) > 0 {
			return true
		}
		return a1.Title() < a2.Title()
	})
	return articles, nil
}

// Load loads articles from the given directory in the given fs
// into the given app.
func Load(a *app.App, fs vfs.VFS, dir string) ([]*article.Article, error) {
	articles, err := List(fs, dir)
	if err != nil {
		return nil, err
	}
	setAppArticles(a, articles)
	return articles, nil
}

// LoadDir works like Load, but loads the articles from the given directory
// in the local filesystem.
func LoadDir(a *app.App, dir string) ([]*article.Article, error) {
	fs, err := vfs.FS(dir)
	if err != nil {
		return nil, err
	}
	return Load(a, fs, "/")
}
