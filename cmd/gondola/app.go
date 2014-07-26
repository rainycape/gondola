package main

import (
	"gnd.la/internal/gen/app"
)

type genAppOptions struct {
	Release bool `help:"Generate release files, otherwise development files are generated"`
}

func genAppCommand(opts *genAppOptions) error {
	a, err := app.Parse(".")
	if err != nil {
		return err
	}
	return a.Gen(opts.Release)
}
