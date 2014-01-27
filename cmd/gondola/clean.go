package main

import (
	"gnd.la/admin"
	"gnd.la/app"
	"gnd.la/log"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	// Since Go fails to rebuild packages which change with
	// different build tags, and the mantainers don't seem
	// interested in fixing it (http://golang.org/issue/3172),
	// we must delete packages that import profile, either direcly
	// or transitively, when cleaning to be sure
	// they're built with the right tags.
	profilePkg = "gnd.la/app/profile"
)

func importsProfile(ctx *build.Context, pkgs map[string]bool, pkg *build.Package) bool {
	if pkg.Goroot {
		return false
	}
	ip := pkg.ImportPath
	imports, ok := pkgs[ip]
	if ok {
		return imports
	}
	for _, v := range pkg.Imports {
		if v == profilePkg {
			imports = true
		}
		s, err := ctx.Import(v, "", 0)
		if err == nil && s != nil {
			// Always call importsProfile for imported pkgs,
			// so we end up walking all the graph.
			imports = importsProfile(ctx, pkgs, s) || imports
		}
	}
	pkgs[ip] = imports
	return imports
}

func clean(dir string) error {
	ctx := build.Default
	ctx.UseAllFiles = true
	pkg, err := ctx.ImportDir(dir, 0)
	if err != nil {
		pkg, err = ctx.Import(dir, "", 0)
	} else {
		// Fix import path, since ImportDir sets ImportPath == dir
		dir, err = filepath.Abs(dir)
		if err != nil {
			return err
		}
		dir = strings.TrimPrefix(dir, filepath.Join(ctx.GOPATH, "src"))
		dir = strings.TrimPrefix(dir, string(filepath.Separator))
		pkg.ImportPath = dir
	}
	if err != nil {
		return err
	}
	pkgs := make(map[string]bool)
	importsProfile(&ctx, pkgs, pkg)
	var toClean []string
	for k, v := range pkgs {
		if v {
			toClean = append(toClean, k)
		}
	}
	args := []string{"clean", "-i", dir, profilePkg}
	args = append(args, toClean...)
	cmd := exec.Command("go", args...)
	log.Debugln("Running", cmdString(cmd))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Clean(ctx *app.Context) {
	var dir string
	ctx.ParseIndexValue(0, &dir)
	if dir == "" {
		dir = "."
	}
	if err := clean(dir); err != nil {
		panic(err)
	}
}

func init() {
	admin.Register(Clean, &admin.Options{
		Help: "Cleans any Gondola packages which use conditional compilation - DO THIS BEFORE BUILDING A BINARY FOR DEPLOYMENT - see golang.org/issue/3172",
	})
}
