package articles

import (
	"fmt"
	"gnd.la/app"
	"gnd.la/util/fileutil"
	"gnd.la/util/generic"
	"gnd.la/util/parseutil"
	"gnd.la/util/stringutil"
	"gnd.la/util/types"
	"gnd.la/util/yaml"
	"io/ioutil"
	"path/filepath"
	"time"
)

const (
	metaFile    = "article.yaml"
	articlesKey = "articles"
)

var (
	defaultNames = []string{"article.html", "article.md", "article.txt"}
)

// AppArticles returns the articles loaded into the given App, or
// nil if no articles were loaded into it.
func AppArticles(a *app.App) []*Article {
	articles, _ := a.Get(articlesKey).([]*Article)
	return articles
}

func setAppArticles(a *app.App, articles []*Article) {
	a.Set(articlesKey, articles)
}

// Article represents an article loaded from a directory.
type Article struct {
	Id         string
	Title      string
	PrevTitles []string
	Synopsis   string
	Created    time.Time
	Updated    time.Time
	Template   string
	Priority   int
}

// Slug returns the article slug, which is derived from the title and is
// used to construct the article URL.
func (a *Article) Slug() string {
	return titleSlug(a.Title)
}

func titleSlug(title string) string {
	return stringutil.SlugN(title, 50)
}

func newArticle(id string, m map[string]interface{}) (*Article, error) {
	art := &Article{Id: id, Priority: 1000}
	if id, ok := m["id"].(string); ok {
		art.Id = id
	}
	if title, ok := m["title"].(string); ok {
		art.Title = title
	}
	if prevTitles, ok := m["prev_titles"]; ok {
		switch x := prevTitles.(type) {
		case string:
			art.PrevTitles = []string{x}
		case []string:
			art.PrevTitles = x
		default:
			return nil, fmt.Errorf("invalid prev_titles value %v of type %T", x, x)
		}
	}
	if synopsis, ok := m["synopsis"].(string); ok {
		art.Synopsis = synopsis
	}
	if template, ok := m["template"].(string); ok {
		art.Template = template
	}
	if p, ok := m["priority"]; ok {
		val, err := types.ToInt(p)
		if err != nil {
			return nil, fmt.Errorf("invalid priority value %v: %s", p, err)
		}
		art.Priority = val
	}
	if v, ok := m["created"].(string); ok {
		t, err := parseutil.DateTime(v)
		if err != nil {
			return nil, fmt.Errorf("invalid created value %v: %s", v, err)
		}
		art.Created = t
	}
	if v, ok := m["updated"].(string); ok {
		t, err := parseutil.DateTime(v)
		if err != nil {
			return nil, fmt.Errorf("invalid created updated %v: %s", v, err)
		}
		art.Updated = t
	}
	return art, nil
}

// List returns the articles found in the given directory.
func List(dir string) ([]*Article, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	dirs, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var articles []*Article
	for _, v := range dirs {
		articleDir := filepath.Join(dir, v.Name())
		meta := filepath.Join(articleDir, metaFile)
		var m map[string]interface{}
		if err := yaml.UnmarshalFile(meta, &m); err != nil {
			return nil, fmt.Errorf("error reading %s: %s", meta, err)
		}
		art, err := newArticle(v.Name(), m)
		if err != nil {
			return nil, fmt.Errorf("error parsing article at %s: %s", articleDir, err)
		}
		if art.Template == "" {
			for _, n := range defaultNames {
				p := filepath.Join(articleDir, n)
				if fileutil.FileExists(p) {
					art.Template = p
					break
				}
			}
			if art.Template == "" {
				return nil, fmt.Errorf("could not find template for article at %s (tried %v)", articleDir, defaultNames)
			}
		} else {
			p := filepath.Join(articleDir, art.Template)
			if !fileutil.FileExists(p) {
				return nil, fmt.Errorf("article template %s does not exist (absolute path %s)", art.Template, p)
			}
			art.Template = p
		}
		articles = append(articles, art)
	}
	generic.SortFunc(articles, func(a1, a2 *Article) bool {
		if a1.Priority < a2.Priority {
			return true
		}
		if a1.Created.Sub(a2.Created) > 0 {
			return true
		}
		return a1.Title < a2.Title
	})
	return articles, nil
}

// Load loasds articles from the given directory into
// the given app.
func Load(a *app.App, dir string) ([]*Article, error) {
	articles, err := List(dir)
	if err != nil {
		return nil, err
	}
	setAppArticles(a, articles)
	return articles, nil
}

// MustLoad works like Load, but panics if there's an error.
func MustLoad(a *app.App, dir string) []*Article {
	articles, err := Load(a, dir)
	if err != nil {
		panic(err)
	}
	return articles
}
