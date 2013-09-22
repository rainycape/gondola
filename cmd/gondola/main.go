package main

import (
	"flag"
	"gnd.la/admin"
	"gnd.la/mux"
)

func main() {
	m := mux.New()
	if !admin.Perform(m) {
		flag.Usage()
	}
}
