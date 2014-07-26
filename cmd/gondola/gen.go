package main

import "gnd.la/internal/gen"

type genOptions struct {
	Genfile string `name:"genfile" help:"Code generation configuration file"`
}

func genCommand(opts *genOptions) error {
	return gen.Gen(".", opts.Genfile)
}
