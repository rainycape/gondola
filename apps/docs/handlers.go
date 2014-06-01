package docs

import (
	"gnd.la/app"
	"gnd.la/apps/docs/doc"
	"gnd.la/log"
	"path"
	"path/filepath"
	"strings"
)

const (
	ListHandlerName    = "docs-list"
	StdListHandlerName = "docs-std-list"
	PackageHandlerName = "docs-package"
	SourceHandlerName  = "docs-source"
)

var (
	PackageTemplateName  = "package.html"
	PackagesTemplateName = "packages.html"
	SourceTemplateName   = "source.html"

	ListHandler    = app.NamedHandler(ListHandlerName, listHandler)
	StdListHandler = app.NamedHandler(StdListHandlerName, stdListHandler)
	PackageHandler = app.NamedHandler(PackageHandlerName, packageHandler)
	SourceHandler  = app.NamedHandler(SourceHandlerName, sourceHandler)
)

type breadcrumb struct {
	Title string
	Href  string
}

type packageGroup struct {
	Title    string
	Packages []*doc.Package
}

func listHandler(ctx *app.Context) {
	var groups []*packageGroup
	for _, gr := range Groups {
		var groupPackages []*doc.Package
		for _, v := range gr.Packages {
			pkgs, err := doc.ImportPackages(packageDir(v))
			if err != nil {
				log.Errorf("error importing %s: %s", v, err)
				continue
			}
			groupPackages = append(groupPackages, pkgs...)
		}
		if len(groupPackages) > 0 {
			groups = append(groups, &packageGroup{
				Title:    gr.Title,
				Packages: groupPackages,
			})
		}
	}
	if len(groups) == 0 {
		ctx.NotFound("no packages to list")
		return
	}
	title := "Package Index"
	data := map[string]interface{}{
		"Header": title,
		"Title":  title,
		"Groups": groups,
	}
	ctx.MustExecute(PackagesTemplateName, data)
}

func stdListHandler(ctx *app.Context) {
	allPkgs, err := doc.ImportPackages(filepath.Join(doc.Context.GOROOT, "src"))
	if err != nil {
		panic(err)
	}
	var pkgs []*doc.Package
	var cmds []*doc.Package
	for _, v := range allPkgs {
		if filepath.Base(v.Dir()) == "cmd" {
			cmds = append(cmds, v)
		} else {
			pkgs = append(pkgs, v)
		}
	}
	title := "Go Standard Library"
	groups := []packageGroup{
		{Title: "Go Standard Library", Packages: pkgs},
		{Title: "Go Commands", Packages: cmds},
	}
	data := map[string]interface{}{
		"Header": title,
		"Title":  title,
		"Groups": groups,
	}
	ctx.MustExecute(PackagesTemplateName, data)
}

func packageHandler(ctx *app.Context) {
	rel := ctx.IndexValue(0)
	if rel[len(rel)-1] == '/' {
		ctx.MustRedirectReverse(true, PackageHandlerName, rel[:len(rel)-1])
		return
	}
	pkg, err := doc.ImportPackage(rel)
	if err != nil {
		panic(err)
	}
	var title string
	var header string
	var distinct bool
	switch {
	case pkg.IsMain():
		title = "Command " + path.Base(rel)
		header = title
	case pkg.IsEmpty():
		prefix := "Directory "
		title = prefix + pkg.ImportPath()
		header = prefix + path.Base(rel)
	default:
		title = "Package " + pkg.ImportPath()
		header = "Package " + pkg.Name()
		distinct = path.Base(pkg.ImportPath()) != pkg.Name()
	}
	breadcrumbs := []*breadcrumb{
		{Title: "Index", Href: ctx.MustReverse(ListHandlerName)},
	}
	if pkg.IsStd() {
		breadcrumbs = append(breadcrumbs, &breadcrumb{
			Title: "std",
			Href:  ctx.MustReverse(PackageHandlerName, "std"),
		})
	}
	for ii := 0; ii < len(rel); {
		var end int
		slash := strings.IndexByte(rel[ii:], '/')
		if slash < 0 {
			end = len(rel)
		} else {
			end = ii + slash
		}
		breadcrumbs = append(breadcrumbs, &breadcrumb{
			Title: rel[ii:end],
			Href:  ctx.MustReverse(PackageHandlerName, rel[:end]),
		})
		ii = end + 1
	}
	data := map[string]interface{}{
		"Header":      header,
		"Title":       title,
		"Breadcrumbs": breadcrumbs,
		"Package":     pkg,
		"Distinct":    distinct,
	}
	ctx.MustExecute("package.html", data)
}

func packageDir(p string) string {
	if strings.IndexByte(p, '.') > 0 {
		// Non std package
		return filepath.Join(doc.Context.GOPATH, "src", p)
	}
	// Std pckage
	if strings.HasPrefix(p, "cmd") {
		return filepath.Join(doc.Context.GOROOT, "src", p)
	}
	return filepath.Join(doc.Context.GOROOT, "src", "pkg", p)
}

func init() {
	doc.App = App
	doc.DocHandlerName = PackageHandlerName
	doc.SourceHandlerName = SourceHandlerName
}
