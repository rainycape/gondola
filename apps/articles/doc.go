// Package articles implements an app for diplaying articles from files.
//
// This application is intended to display a collection of articles in
// which each article is composed of a directory with the following structure:
//
//  - a metadata file, named article.yaml
//  - an article file, by default the first found of article.html, article.txt and article.md
//
// Note that in order to convert .md templates to HTML, your application must
// import gnd.la/template/markdown.
//
// The metadata file is a YAML file with the following keys:
//
//  - title (required): Indicates the article title and is the basis for deriving
//	the article URL.
//  - synopsis (optional): A small text about the article shown in article listings.
//  - prev_titles (optional): Previous titles this article had, used for redirecting
//	old URLs to the new URL.
//  - created (optional): Article creation date (parsed with parseutil.DateTime).
//  - updated (optional): Last article update (parsed with parseutil.DateTime).
//  - template (optional): The template filename for the article. If not provided, article.html,
//	article.txt and article.md are tried in that order.
//  - priotity (optional, default 1000): When listing the articles, the ones with lower priority
//	are shown first.
//
// The typical usage of this application is as follows:
//
//  myapp.Include("/articles/", articles.App, "articles-base.html")
//  articles.MustLoad(articles.App, pathutil.Relative("articles"))
//
// Also, this app can be included multiple times by cloning it:
//
//  articlesApp := articles.App.Clone()
//  myapp.Include("/articles/", articlesApp, "articles-base.html")
//  articles.MustLoad(articlesApp, pathutil.Relative("articles")) // Load articles from the "articles" dir
//
//  // included a second time
//  tutorialsApp := articles.App.Clone()
//  tutorialsApp.SetName("Tutorials") // Set the listing title to Tutorials
//  myapp.Include("/tutorials/", tutorialsApp, "articles-base.html")
//  articles.MustLoad(tutorialsApp, pathutil.Relative("tutorials")) // Load articles from the "tutorials" dir
package articles
