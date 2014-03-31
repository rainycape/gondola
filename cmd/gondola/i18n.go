package main

import (
	"gnd.la/admin"
	"gnd.la/app"
	"gnd.la/i18n/messages"
	"gnd.la/i18n/po"
	"os"
	"path/filepath"
	"strings"
)

func MakeMessages(ctx *app.Context) {
	m, err := messages.Extract(".", nil)
	if err != nil {
		panic(err)
	}
	var out string
	ctx.ParseParamValue("o", &out)
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		panic(err)
	}
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := messages.Write(f, m); err != nil {
		panic(err)
	}
}

func CompileMessages(ctx *app.Context) {
	var poFiles []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".po" {
			poFiles = append(poFiles, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	pos := make([]*po.Po, len(poFiles))
	for ii, v := range poFiles {
		p, err := po.ParseFile(v)
		if err != nil {
			panic(err)
		}
		pos[ii] = p
	}
	var out string
	var defCtx string
	ctx.ParseParamValue("o", &out)
	ctx.ParseParamValue("ctx", &defCtx)
	opts := &messages.CompileOptions{DefaultContext: defCtx}
	if err := messages.Compile(out, pos, opts); err != nil {
		panic(err)
	}
}

func init() {
	admin.Register(MakeMessages, &admin.Options{
		Help: "Generate strings files from the current package (including its non-package subdirectories, like templates)",
		Flags: admin.Flags(
			admin.StringFlag("o", "_messages"+string(filepath.Separator)+"messages.pot", "Output filename. If empty, messages are printed to stdout."),
		),
	})
	admin.Register(CompileMessages, &admin.Options{
		Help: "Compiles all po files from the current directory and its subdirectories",
		Flags: admin.Flags(
			admin.StringFlag("o", "messages.go", "Output filename. Can't be empty."),
			admin.StringFlag("ctx", "", "Default context for messages without it."),
		),
	})
}
