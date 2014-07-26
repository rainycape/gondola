package main

import (
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gnd.la/log"
)

var (
	staticFlags = map[string]bool{
		"ignore":  true,
		"IGNORE":  true,
		"none":    true,
		"NONE":    true,
		"windows": true,
		"linux":   true,
		"darwin":  true,
		"freebsd": true,
		"openbsd": true,
		"netbsd":  true,
		"amd64":   true,
		"386":     true,
		"arm":     true,
		"cgo":     true,
		"go1.1":   true,
		"go1.2":   true,
		"go1.3":   true,
	}
)

// Since Go fails to rebuild packages which change with
// different build tags, and the mantainers don't seem
// interested in fixing it (http://golang.org/issue/3172),
// we must delete packages that depend on builds tags,
// either direcly or transitively, when cleaning to be sure
// they're built with the right tags.
func flagsMayVary(flags []string) bool {
	for _, v := range flags {
		if !staticFlags[v] {
			return true
		}
	}
	return false
}

func usesTags(ctx *build.Context, pkgs map[string]bool, pkg *build.Package) bool {
	if pkg.Goroot {
		return false
	}
	ip := pkg.ImportPath
	uses, ok := pkgs[ip]
	if ok {
		return uses
	}
	uses = flagsMayVary(pkg.AllTags)
	pkgs[ip] = uses
	for _, v := range pkg.Imports {
		s, err := ctx.Import(v, "", 0)
		if err == nil && s != nil {
			// Always call usesTags for imported pkgs,
			// so we end up walking all the graph.
			uses = usesTags(ctx, pkgs, s) || uses
		}
	}
	pkgs[ip] = uses
	return uses
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
	usesTags(&ctx, pkgs, pkg)
	var toClean []string
	for k, v := range pkgs {
		if v {
			toClean = append(toClean, k)
		}
	}
	args := []string{"clean", "-i"}
	if !pkgs[dir] {
		args = append(args, dir)
	}
	args = append(args, toClean...)
	cmd := exec.Command("go", args...)
	log.Debugln("Running", cmdString(cmd))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func cleanCommand(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	return clean(dir)
}
