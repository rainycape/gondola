package docs

import (
	"path"
	"strings"

	"gnd.la/app"
	"gnd.la/apps/docs/doc"
	"gnd.la/log"
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

func docContext(ctx *app.Context) doc.Context {
	return doc.DefaultContext
}

func listHandler(ctx *app.Context) {
	var groups []*packageGroup
	dctx := docContext(ctx)
	for _, gr := range Groups {
		var groupPackages []*doc.Package
		for _, v := range gr.Packages {
			pkgs, err := dctx.ImportPackages(packageDir(dctx, v))
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
	dctx := docContext(ctx)
	allPkgs, err := dctx.ImportPackages(dctx.Join(dctx.GOROOT, "src"))
	if err != nil {
		panic(err)
	}
	var pkgs []*doc.Package
	var cmds []*doc.Package
	for _, v := range allPkgs {
		if dctx.Base(v.Dir()) == "cmd" {
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
	dctx := docContext(ctx)
	rel := ctx.IndexValue(0)
	if rel[len(rel)-1] == '/' {
		ctx.MustRedirectReverse(true, PackageHandlerName, rel[:len(rel)-1])
		return
	}
	pkg, err := dctx.ImportPackage(rel)
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
		header = prefix + path.Base(rel)
		if pkg.ImportPath() != "" {
			title = prefix + pkg.ImportPath()
		} else {
			title = header
		}
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

func packageDir(dctx doc.Context, p string) string {
	if strings.IndexByte(p, '.') > 0 {
		// Non std package
		return dctx.Join(dctx.GOPATH, "src", p)
	}
	// Std pckage
	if strings.HasPrefix(p, "cmd") {
		return dctx.Join(dctx.GOROOT, "src", p)
	}
	return dctx.Join(dctx.GOROOT, "src", "pkg", p)
}

func init() {
	doc.DefaultContext.App = App
	doc.DefaultContext.DocHandlerName = PackageHandlerName
	doc.DefaultContext.SourceHandlerName = SourceHandlerName
}
