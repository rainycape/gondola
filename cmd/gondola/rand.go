package main

import (
	"fmt"

	"gnd.la/util/stringutil"
)

const (
	defaultRandomLength = 64
)

type randomStringOptions struct {
	Length int `help:"Length of the generated random string"`
}

func randomStringCommand(opts *randomStringOptions) {
	fmt.Println(stringutil.RandomPrintable(opts.Length))
}
