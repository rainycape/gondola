package main

import (
	"flag"
	"gnd.la/admin"
	"gnd.la/log"
	"gnd.la/mux"
)

func main() {
	quiet := flag.Bool("q", false, "Be quiet")
	flag.Parse()
	if !*quiet {
		log.SetLevel(log.LDebug)
	}
	m := mux.New()
	if !admin.Perform(m) {
		flag.Usage()
	}
}
