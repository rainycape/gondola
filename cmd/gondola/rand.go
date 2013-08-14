package main

import (
	"fmt"
	"gondola/admin"
	"gondola/mux"
	"gondola/util"
)

const (
	defaultSecretLength = 64
)

func RandomString(ctx *mux.Context) {
	var length int
	ctx.ParseParamValue("length", &length)
	fmt.Println(util.RandomPrintableString(length))
}

func init() {
	admin.Register(RandomString, &admin.Options{
		Help:  "Generates a random string suitable for use as the mux secret",
		Flags: admin.Flags(admin.IntFlag("length", defaultSecretLength, "Length of the generated secret string")),
	})
}
