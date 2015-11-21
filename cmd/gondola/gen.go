package main

import (
	"gnd.la/internal/gen"

	"github.com/rainycape/command"
)

type genOptions struct {
	Genfile string `name:"genfile" help:"Code generation configuration file"`
}

func genCommand(_ *command.Args, opts *genOptions) error {
	return gen.Gen(".", opts.Genfile)
}
