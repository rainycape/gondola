package main

import (
	"gnd.la/admin"
	"gnd.la/app"
	"gnd.la/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

func RmGen(ctx *app.Context) {
	var dir string
	ctx.ParseIndexValue(0, &dir)
	if dir == "" {
		dir = "."
	}
	re := regexp.MustCompile("(?i).+\\.gen\\..+")
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() && re.MatchString(path) {
			log.Infof("Removing %s", path)
			if err := os.Remove(path); err != nil {
				panic(err)
			}
			dir := filepath.Dir(path)
			if infos, err := ioutil.ReadDir(dir); err == nil && len(infos) == 0 {
				log.Infof("Removing empty dir %s", dir)
				if err := os.Remove(dir); err != nil {
					panic(err)
				}
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func init() {
	admin.Register(RmGen, &admin.Options{
		Help: "Remove Gondola generated files (identified by *.gen.*)",
	})
}
