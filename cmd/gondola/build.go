package main

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

const (
	buildHelp = `gondola build is a wrapper around go build.

Before building a package, gondola build checks that all its dependencies
exist and automatically downloads the missing ones.

The -go option can be used to set the go command that will be called. All the
remaining options are passed to go build unchanged.`
)

type buildOptions struct {
	Go         string `help:"Command to run the go tool"`
	Race       bool   `help:"Enable data race detection"`
	Print      bool   `name:"x" help:"Print the commands"`
	Verbose    bool   `name:"v" help:"Print the names of packages as they are compiled"`
	CCFlags    string `help:"Arguments to pass on each 5c, 6c, or 8c compiler invocation"`
	Compiler   string `help:"Name of compiler to use, as in runtime.Compiler (gccgo or gc)"`
	GccGoFlags string `help:"Arguments to pass on each gccgo compiler/linker invocation"`
	GcFlags    string `help:"Arguments to pass on each 5g, 6g, or 8g compiler invocation"`
	LDFlags    string `help:"Arguments to pass on each 5l, 6l, or 8l linker invocation"`
	Tags       string `help:"A list of build tags to consider satisfied during the build"`
}

func runGoBuild(pkg string, opts *buildOptions) error {
	args := []string{"build"}
	if opts.Race {
		args = append(args, "-race")
	}
	if opts.Print {
		args = append(args, "-x")
	}
	if opts.Verbose {
		args = append(args, "-v")
	}
	stringOpts := []string{"CCFlags", "Compiler", "GccGoFlags", "GcFlags", "LDFlags", "Tags"}
	val := reflect.ValueOf(opts).Elem()
	for _, field := range stringOpts {
		fieldVal := val.FieldByName(field)
		if s := fieldVal.String(); s != "" {
			args = append(args, "-"+strings.ToLower(field), s)
		}
	}
	if pkg != "" && pkg != "." {
		args = append(args, pkg)
	}
	cmd := exec.Command(opts.Go, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if opts.Print || opts.Verbose {
		fmt.Printf("running %s %s\n", opts.Go, strings.Join(args, " "))
	}
	return cmd.Run()
}

func splitString(s string) []string {
	split := strings.Split(s, ",")
	var res []string
	for _, v := range split {
		s := strings.TrimSpace(v)
		if s != "" {
			res = append(res, s)
		}
	}
	return res
}

func importPackage(pkg string, opts *buildOptions) (*build.Package, error) {
	ctx := build.Default
	if opts.Compiler != "" {
		ctx.Compiler = opts.Compiler
	}
	if opts.Tags != "" {
		ctx.BuildTags = splitString(opts.Tags)
	}
	p, err := ctx.ImportDir(pkg, 0)
	if err != nil {
		p, err = ctx.Import(pkg, "", 0)
	}
	return p, err
}

func checkImports(pkg string, opts *buildOptions, cache map[string]error) error {
	if pkg == "C" {
		return nil
	}
	if e, ok := cache[pkg]; ok {
		return e
	}
	if opts.Verbose && pkg != "" && pkg != "." {
		fmt.Printf("checking package %s\n", pkg)
	}
	var p *build.Package
	var err error
	p, err = importPackage(pkg, opts)
	// No better way to test, since sometimes returned error is just an errors.errorString
	if err != nil && (os.IsNotExist(err) || strings.Contains(err.Error(), "cannot find package")) {
		// Package does not exist, go get it
		args := []string{"get"}
		if opts.Verbose {
			args = append(args, "-v")
		}
		cmd := exec.Command("go", args...)
		// Send all output to os.Stdout, leave os.Stderr
		// only for errors
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout
		if opts.Verbose {
			fmt.Printf("running go %s\n", strings.Join(args, " "))
		}
		if err = cmd.Run(); err != nil {
			err = fmt.Errorf("could not download %s", pkg)
		}
		if err == nil {
			// Check this package again, to get its imports
			return checkImports(pkg, opts, cache)
		}
	}
	cache[pkg] = err
	if p != nil && !p.Goroot {
		for _, v := range p.Imports {
			if err = checkImports(v, opts, cache); err != nil {
				break
			}
		}
	}
	return err
}

// don't use gnd.la/log in this command, since it prints to os.Stderr
// and it gets the log output mixed with any potential errors from go build

func buildCommand(args []string, opts *buildOptions) error {
	if len(args) == 0 {
		args = []string{"."}
	}
	if opts.Go == "" {
		opts.Go = "go"
	}
	cache := make(map[string]error)
	for _, v := range args {
		if err := checkImports(v, opts, cache); err != nil {
			return fmt.Errorf("error getting %s dependencies: %s", v, err)
		}
		if err := runGoBuild(v, opts); err != nil {
			return fmt.Errorf("error building %s: %s", v, err)
		}
	}
	return nil
}
