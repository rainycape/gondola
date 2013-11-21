package assets

import (
	"code.google.com/p/go.exp/fsnotify"
	"gnd.la/log"
	"os"
	"path/filepath"
)

type WatchFunc func(string, bool)

type Watcher struct {
	watcher *fsnotify.Watcher
	Dir     string
	Func    WatchFunc
}

func NewWatcher(dir string, f WatchFunc) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		watcher: watcher,
		Dir:     dir,
		Func:    f,
	}, nil
}

func (w *Watcher) Watch() error {
	flags := uint32(fsnotify.FSN_DELETE | fsnotify.FSN_MODIFY)
	err := filepath.Walk(w.Dir, func(path string, info os.FileInfo, err error) error {
		if info != nil && info.IsDir() {
			return w.watcher.WatchFlags(path, flags)
		}
		return nil
	})
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case ev := <-w.watcher.Event:
				w.Func(ev.Name[len(w.Dir)+1:], ev.IsDelete())
			case err := <-w.watcher.Error:
				log.Warningf("Error watching %s: %s", w.Dir, err)
			}
		}
	}()
	return err
}

func (w *Watcher) Close() error {
	return w.watcher.Close()
}
