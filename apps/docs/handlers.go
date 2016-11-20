package docs

import (
	"path"
	"strings"

	"gnd.la/app"
	"gnd.la/app/reusableapp"
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
)

type breadcrumb struct {
	Title string
	Href  string
}

type packageGroup struct {
	Title    string
	Packages []*doc.Package
}

func ListHandler(ctx *app.Context) {
	var groups []*packageGroup
	dctx := getEnvironment(ctx)
	for _, gr := range appDocGroups(ctx) {
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

func StdListHandler(ctx *app.Context) {
	dctx := getEnvironment(ctx)
	allPkgs, err := dctx.ImportPackages(dctx.Join(dctx.Context.GOROOT, "src"))
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

func PackageHandler(ctx *app.Context) {
	dctx := getEnvironment(ctx)
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

func packageDir(dctx *doc.Environment, p string) string {
	if strings.IndexByte(p, '.') > 0 {
		// Non std package
		return dctx.Join(dctx.Context.GOPATH, "src", p)
	}
	// Std package
	if strings.HasPrefix(p, "cmd") {
		return dctx.Join(dctx.Context.GOROOT, "src", p)
	}
	// pkg was removed from path to the source around go 1.4
	stdSrc := dctx.Join(dctx.Context.GOROOT, "src", "pkg")
	if !dctx.IsDir(stdSrc) {
		stdSrc = dctx.Join(dctx.Context.GOROOT, "src")
	}
	return dctx.Join(stdSrc, p)
}

func getEnvironment(ctx *app.Context) *doc.Environment {
	data, _ := reusableapp.Data(ctx).(*appData)
	if data != nil {
		return data.Environment
	}
	return nil
}
