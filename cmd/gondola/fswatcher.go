package main

import (
	"go/build"
	"time"

	"gnd.la/log"
	"gnd.la/util/generic"

	"gopkg.in/fsnotify.v1"
)

type fsWatcher struct {
	watcher     *fsnotify.Watcher
	Changed     func(string)
	IsValidFile func(string) bool
}

func newFSWatcher() (*fsWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watcher := &fsWatcher{watcher: w}
	go watcher.watch()
	return watcher, nil
}

func (w *fsWatcher) Add(path string) error {
	return w.watcher.Add(path)
}

func (w *fsWatcher) Remove(path string) error {
	return w.watcher.Remove(path)
}

func (w *fsWatcher) Close() {
	if w.watcher != nil {
		w.watcher.Close()
		w.watcher = nil
	}
}

func (w *fsWatcher) AddPackages(pkgs []*build.Package) error {
	paths := generic.Map(pkgs, func(pkg *build.Package) string { return pkg.Dir }).([]string)
	for _, p := range paths {
		if err := w.Add(p); err != nil {
			return err
		}
	}
	return nil
}

func (w *fsWatcher) watch() {
	var t *time.Timer
	for {
		select {
		case ev, ok := <-w.watcher.Events:
			if !ok {
				// Closed
				return
			}
			if ev.Op == fsnotify.Chmod {
				break
			}
			if ev.Op == fsnotify.Remove {
				// It seems the Watcher stops watching a file
				// if it receives a DELETE event for it. For some
				// reason, some editors generate a DELETE event
				// for a file when saving it, so we must watch the
				// file again. Since fsnotify is in exp/ and its
				// API might change, remove the watch first, just
				// in case.
				w.watcher.Remove(ev.Name)
				w.watcher.Add(ev.Name)
			}
			if w.isValidFile(ev.Name) {
				if t != nil {
					t.Stop()
				}
				name := ev.Name
				t = time.AfterFunc(50*time.Millisecond, func() {
					w.changed(name)
				})
			}
		case err := <-w.watcher.Errors:
			if err == nil {
				// Closed
				return
			}
			log.Errorf("Error watching: %s", err)
		}
	}
}

func (w *fsWatcher) changed(path string) {
	if w.Changed != nil {
		w.Changed(path)
	}
}

func (w *fsWatcher) isValidFile(path string) bool {
	return w.IsValidFile != nil && w.IsValidFile(path)
}
