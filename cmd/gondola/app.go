package main

import (
	"gnd.la/admin"
	"gnd.la/app"
	gapp "gnd.la/gen/app"
)

func GenApp(ctx *app.Context) {
	var release bool
	ctx.ParseParamValue("release", &release)
	a, err := gapp.Parse(".")
	if err != nil {
		panic(err)
	}
	if err := a.Gen(release); err != nil {
		panic(err)
	}
}

func init() {
	admin.Register(GenApp, &admin.Options{
		Help: "Generate boilerplate code for a Gondola app from the app.yaml file",
		Flags: admin.Flags(
			admin.BoolFlag("release", false, "Generate release files"),
		),
	})
}
