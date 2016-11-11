package template

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"path"

	"gnd.la/log"
	"gnd.la/template/assets"
)

func executeAsset(t *Template, p *Template, vars VarMap, m *assets.Manager, asset *assets.Asset) (string, error) {
	name := asset.TemplateName()
	log.Debugf("executing asset template %s (from %s)", name, t.name)
	tmpl := New(m.VFS(), nil)
	tmpl.addFuncMap(t.funcMap, true)
	if p != nil {
		tmpl.addFuncMap(p.funcMap, true)
	}
	if err := tmpl.Parse(name); err != nil {
		return "", err
	}
	if err := tmpl.Compile(); err != nil {
		return "", err
	}
	if p != nil && p != t {
		ns := t.namespaceIn(p)
		if ns != "" {
			vars = vars.unpack(ns)
		}
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteContext(&buf, nil, nil, vars); err != nil {
		return "", err
	}
	f, err := m.Load(name)
	if err != nil {
		return "", err
	}
	h := fnv.New32a()
	// Writing to a hash.Hash never fails
	// Include the initial and the final
	// asset in the hash.
	io.Copy(h, f)
	f.Close()
	h.Write(buf.Bytes())
	hash := hex.EncodeToString(h.Sum(nil))
	ext := path.Ext(name)
	nonExt := name[:len(name)-len(ext)]
	out := fmt.Sprintf("%s.gen.%s%s", nonExt, hash, ext)
	// Check if the file already exists and has the same
	// contents. This prevents failures when runnin on
	// App Engine, since writes are not allowed.
	if prev, err := m.Load(out); err == nil {
		defer prev.Close()
		if data, err := ioutil.ReadAll(prev); err == nil {
			if bytes.Equal(data, buf.Bytes()) {
				// File already up to date
				return out, nil
			}
		}
	}
	w, err := m.Create(out, true)
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
