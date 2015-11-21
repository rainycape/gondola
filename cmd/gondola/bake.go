package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"path/filepath"
	"strings"

	"github.com/rainycape/command"

	"gnd.la/internal/gen/genutil"
	"gnd.la/internal/vfsutil"
	"gnd.la/log"
)

type bakeOptions struct {
	Dir        string `help:"Root directory with the files to bake"`
	Name       string `help:"Variable name of the generated VFS"`
	Out        string `name:"o" help:"Output filename. If empty, output is printed to standard output"`
	Extensions string `name:"ext" help:"Additional extensions (besides html, css and js) to include, separated by commas"`
}

func bakeCommand(_ *command.Args, opts *bakeOptions) error {
	extensions := []string{".html", ".css", ".js"}
	if opts.Dir == "" {
		return errors.New("dir can't be empty")
	}
	if opts.Name == "" {
		return errors.New("name can't be empty")
	}
	extensions = append(extensions, strings.Split(opts.Extensions, ",")...)
	var buf bytes.Buffer
	odir := filepath.Dir(opts.Out)
	p, err := build.ImportDir(odir, 0)
	if err == nil {
		buf.WriteString(fmt.Sprintf("package %s\n", p.Name))
	}
	buf.WriteString("import \"gnd.la/internal/vfsutil\"\n")
	buf.WriteString(genutil.AutogenString())
	fmt.Fprintf(&buf, "var %s = ", opts.Name)
	if err := vfsutil.BakedFS(&buf, opts.Dir, extensions); err != nil {
		return err
	}
	if err := genutil.WriteAutogen(opts.Out, buf.Bytes()); err != nil {
		return err
	}
	log.Debugf("Assets written to %s (%d bytes)", opts.Out, buf.Len())
	return nil
}
