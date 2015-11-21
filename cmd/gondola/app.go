package main

import (
	"gnd.la/internal/gen/app"

	"github.com/rainycape/command"
)

type genAppOptions struct {
	Release bool `help:"Generate release files, otherwise development files are generated"`
}

func genAppCommand(_ *command.Args, opts *genAppOptions) error {
	a, err := app.Parse(".")
	if err != nil {
		return err
	}
	return a.Gen(opts.Release)
}
