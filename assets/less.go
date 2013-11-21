package assets

import (
	"bytes"
	"fmt"
	"gnd.la/log"
	"gnd.la/util/hashutil"
	"io"
	"path"
)

func lessCompile(m Manager, name string, options Options) (string, error) {
	f, _, err := m.Load(name)
	if err != nil {
		return "", err
	}
	fnv := hashutil.Fnv32a(f)
	out := fmt.Sprintf("%s.%s.css", name[:len(name)-len(path.Ext(name))], fnv)
	if _, _, err := m.Load(out); err == nil {
		log.Debugf("%s already compiled to %s", name, out)
		return out, nil
	}
	f.Seek(0, 0)
	var buf bytes.Buffer
	log.Debugf("compiling %s to %s", name, out)
	if _, _, err := reducer("less", &buf, f); err != nil {
		return "", err
	}
	w, err := m.Create(out)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(w, bytes.NewReader(buf.Bytes())); err != nil {
		w.Close()
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return out, nil
}
