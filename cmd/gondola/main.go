package main

import (
	"flag"
	"gondola/admin"
	"gondola/mux"
)

func main() {
	m := mux.New()
	if !admin.Perform(m) {
		flag.Usage()
	}
}
