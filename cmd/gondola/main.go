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
	admin.Remove("cat-file")
	admin.Remove("make-assets")
	if !admin.Execute(a) {
		flag.Usage()
	}
}
