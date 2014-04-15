package project

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"gnd.la/util/yaml"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	metaFile = "_template.yaml"
)

type Template struct {
	Name        string `json:"name"`
	Description string `json:"desc"`
	URL         string `json:"url"`
	Version     int    `json:"v"`
	dir         string
}

func (t *Template) addFiles(tw *tar.Writer, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := strings.TrimPrefix(path, dir)
		if name != "" {
			if name[0] == filepath.Separator {
				name = name[1:]
			}
			if name == "_template.yaml" {
				return nil
			}
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			if dir, _ := filepath.Split(name); dir != "" {
				hdr.Name = filepath.ToSlash(filepath.Join(dir, hdr.Name))
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if !info.IsDir() {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				if _, err := io.Copy(tw, f); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (t *Template) addFilesFromSibling(tw *tar.Writer, name string) error {
	return t.addFiles(tw, filepath.Join(filepath.Dir(t.dir), name))
}

func (t *Template) Data(gae bool) ([]byte, error) {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	tw := tar.NewWriter(zw)
	if err := t.addFiles(tw, t.dir); err != nil {
		return nil, err
	}
	if gae {
		if err := t.addFilesFromSibling(tw, "_appengine"); err != nil {
			return nil, err
		}
	}
	if filepath.Base(t.dir) != "blank" {
		if err := t.addFilesFromSibling(tw, "_common"); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func LoadTemplates(dir string) ([]*Template, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var templates []*Template
	for _, v := range files {
		if s := v.Name()[0]; s == '.' || s == '_' {
			continue
		}
		tmplDir := filepath.Join(dir, v.Name())
		var tmpl Template
		meta := filepath.Join(tmplDir, metaFile)
		if err := yaml.UnmarshalFile(meta, &tmpl); err != nil {
			return nil, err
		}
		tmpl.dir = tmplDir
		tmpl.Name = v.Name()
		templates = append(templates, &tmpl)
	}
	return templates, nil
}
