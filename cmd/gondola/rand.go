package main

import (
	"fmt"

	"gnd.la/util/stringutil"

	"github.com/rainycape/command"
)

const (
	defaultRandomLength = 64
)

type randomStringOptions struct {
	Length int `help:"Length of the generated random string"`
}

func randomStringCommand(_ *command.Args, opts *randomStringOptions) {
	fmt.Println(stringutil.RandomPrintable(opts.Length))
}
