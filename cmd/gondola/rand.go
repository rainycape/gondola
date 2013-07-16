package main

import (
	"fmt"
	"gondola/admin"
	"gondola/mux"
	"gondola/util"
)

const (
	secretLength = 64
)

func RandomString(ctx *mux.Context) {
	fmt.Println(util.RandomPrintableString(secretLength))
}

func init() {
	admin.Register(
		admin.H(RandomString, "Generates a random string suitable for use as the mux secret"),
	)
}
