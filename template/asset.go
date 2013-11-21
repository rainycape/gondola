package template

import (
	"bytes"
	"fmt"
	"gnd.la/log"
	"gnd.la/util/hashutil"
	"io/ioutil"
	"path"
	"text/template"
)

func executeAsset(t *Template, name string) (string, error) {
	log.Debugf("executing asset template %s", name)
	f, _, err := t.AssetsManager.Load(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	tmpl := template.New(name)
	if _, err := tmpl.Parse(string(data)); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, t.vars); err != nil {
		return "", err
	}
	ext := path.Ext(name)
	nonExt := name[:len(name)-len(ext)]
	f.Seek(0, 0)
	out := fmt.Sprintf("%s.%s%s", nonExt, hashutil.Fnv32a(f), ext)
	w, err := t.AssetsManager.Create(out, true)
	if err != nil {
		return "", err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return out, nil
}
