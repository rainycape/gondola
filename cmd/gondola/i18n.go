package main

import (
	"os"
	"path/filepath"
	"strings"

	"gnd.la/i18n/messages"
	"gnd.la/i18n/po"
)

type makeMessagesOptions struct {
	Out string `name:"o" help:"Output filename. If empty, messages are printed to stdout."`
}

func makeMessagesCommand(opts *makeMessagesOptions) error {
	m, err := messages.Extract(".", nil)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(opts.Out), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(opts.Out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := messages.Write(f, m); err != nil {
		return err
	}
	return f.Close()
}

type compileMessagesOptions struct {
	Out     string `name:"o" help:"Output filename. Can't be empty."`
	Context string `name:"ctx" help:"Default context for messages without it."`
}

func compileMessagesCommand(opts *compileMessagesOptions) error {
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
		return err
	}
	pos := make([]*po.Po, len(poFiles))
	for ii, v := range poFiles {
		p, err := po.ParseFile(v)
		if err != nil {
			return err
		}
		pos[ii] = p
	}
	copts := &messages.CompileOptions{DefaultContext: opts.Context}
	return messages.Compile(opts.Out, pos, copts)
}
