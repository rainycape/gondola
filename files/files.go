package files

import (
	"exp/inotify"
	"fmt"
	"hash/adler32"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
)

var dirs = make(map[string]string)
var hashes = make(map[string]string)
var mutex = sync.RWMutex{}

func watchDir(dir string, f func(string, string)) {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	mask := inotify.IN_DELETE | inotify.IN_MODIFY
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			err = watcher.AddWatch(path, mask)
			if err != nil {
				panic(err)
			}
		}
		return nil
	})
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				f(dir, ev.Name)
			case err := <-watcher.Error:
				log.Printf("Error watching %s: %s", dir, err)
			}
		}
	}()
}

func StaticFilesHandler(prefix string, dir string) func(http.ResponseWriter, *http.Request) {
	prefixLength := len(prefix)
	dirs[prefix] = dir
	dirLen := len(dir)
	watchDir(dir, func(dir string, filename string) {
		relative := filename[dirLen:]
		url := getStaticFileUrl(prefix, relative)
		hash, err := GetFileHash(filename)
		if hash != "" && err == nil {
			mutex.Lock()
			hashes[url] = hash
			mutex.Unlock()
		} else {
			delete(hashes, "url")
		}
	})
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if !(p[1] == 'f' || p[1] == 'r') && !(p == "/favicon.ico" || p == "/robots.txt") {
			p = p[prefixLength:]
		}
		f, err := os.Open(filepath.Join(dir, filepath.FromSlash(path.Clean("/"+p))))
		if err != nil {
			log.Printf("Error serving %s: %s\n", p, err)
			return
		}
		defer f.Close()
		d, err := f.Stat()
		if err != nil {
			log.Printf("Error serving %s: %s\n", p, err)
			return
		}
		if r.URL.RawQuery != "" {
			w.Header().Set("Expires", "Thu, 31 Dec 2037 23:55:55 GMT")
			w.Header().Set("Cache-Control", "max-age=315360000")
		}
		http.ServeContent(w, r, p, d.ModTime(), f)
	}
}

func getStaticFileUrl(prefix string, name string) string {
	return path.Clean(prefix + name)
}

func StaticFileUrl(prefix string, name string) string {
	url := getStaticFileUrl(prefix, name)
	mutex.RLock()
	hash, ok := hashes[url]
	mutex.RUnlock()
	if !ok {
		fileDir := dirs[prefix]
		filePath := path.Clean(path.Join(fileDir, name))
		var err error
		hash, err = GetFileHash(filePath)
		if err == nil {
			mutex.Lock()
			hashes[url] = hash
			mutex.Unlock()
		}
	}
	if hash != "" {
		url += "?v=" + hash
	}
	return url
}

func GetFileHash(filename string) (string, error) {
	b, err := ioutil.ReadFile(filename)
	if err == nil {
		value := adler32.Checksum(b)
		return fmt.Sprintf("%04x", value)[:4], nil
	}
	return "", err
}
