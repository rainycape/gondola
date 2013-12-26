package main

import (
	"fmt"
	"gnd.la/admin"
	"gnd.la/app"
	"gnd.la/util"
)

const (
	defaultSecretLength = 64
)

func RandomString(ctx *app.Context) {
	var length int
	ctx.ParseParamValue("length", &length)
	fmt.Println(util.RandomPrintableString(length))
}

func init() {
	admin.Register(RandomString, &admin.Options{
		Help:  "Generates a random string suitable for use as the app secret",
		Flags: admin.Flags(admin.IntFlag("length", defaultSecretLength, "Length of the generated secret string")),
	})
}
