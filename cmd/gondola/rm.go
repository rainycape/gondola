package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"gnd.la/log"
)

func rmGenCommand(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	re := regexp.MustCompile("(?i).+\\.gen\\..+")
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() && re.MatchString(path) {
			log.Infof("Removing %s", path)
			if err := os.Remove(path); err != nil {
				return err
			}
			dir := filepath.Dir(path)
			if infos, err := ioutil.ReadDir(dir); err == nil && len(infos) == 0 {
				log.Infof("Removing empty dir %s", dir)
				if err := os.Remove(dir); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
