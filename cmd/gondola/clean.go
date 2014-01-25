package main

import (
	"gnd.la/admin"
	"gnd.la/app"
	"os/exec"
)

var (
	// These are the Gondola packages which use conditional
	// compilation, either via build tags or by branching
	// on constants defined via build tags in other packages.
	// Since Go fails to rebuild packages which change with
	// different build tags, and the mantainers don't seem
	// interested in fixing it (http://golang.org/issue/3172),
	// we must delete these packages when cleaning to be sure
	// they're built with the right tags.
	conditional = []string{
		"gnd.la/app",
		"gnd.la/app/profile",
		"gnd.la/cache",
		"gnd.la/orm/driver/sql",
		"gnd.la/orm",
		"gnd.la/template",
	}
)

func clean(dir string) error {
	args := []string{"clean", "-i", dir}
	args = append(args, conditional...)
	return exec.Command("go", args...).Run()
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
