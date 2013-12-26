package main

import (
	"flag"
	"gnd.la/admin"
	"gnd.la/app"
	"gnd.la/log"
)

func main() {
	quiet := flag.Bool("q", false, "Be quiet")
	flag.Parse()
	if !*quiet {
		log.SetLevel(log.LDebug)
	}
	a := app.New()
	if !admin.Perform(a) {
		flag.Usage()
	}
}
