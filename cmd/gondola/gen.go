package main

import (
	"gnd.la/admin"
	"gnd.la/gen/json"
	"gnd.la/mux"
)

func GenJson(ctx *mux.Context) {
	if err := json.Gen(".", nil); err != nil {
		panic(err)
	}
}

func init() {
	admin.Register(GenJson, &admin.Options{
		Help: "Generate JSONWriter methods for the exported types in the current directory",
	})
}
