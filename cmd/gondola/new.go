package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"text/template"

	"gnd.la/internal/project"
	"gnd.la/log"
	"gnd.la/net/urlutil"
	"gnd.la/util/fileutil"
	"gnd.la/util/generic"
	"gnd.la/util/stringutil"
)

const (
	serverUrl = "http://www.gondolaweb.com/api/v1"
)

func getAvailableTemplates() ([]*project.Template, error) {
	ep := serverUrl + "/templates"
	resp, err := http.Get(ep)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpect HTTP response code %d", resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	var templates []*project.Template
	if err := dec.Decode(&templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func getTemplateReader(url string, gae bool) (*tar.Reader, error) {
	ep, err := urlutil.Join(serverUrl, url)
	if err != nil {
		return nil, err
	}
	if gae {
		ep += "?gae=1"
	}
	resp, err := http.Get(ep)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpect HTTP response code %d", resp.StatusCode)
	}
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	data, err := ioutil.ReadAll(zr)
	if err != nil {
		return nil, err
	}
	return tar.NewReader(bytes.NewReader(data)), nil
}

func isEmptyDir(dir string) bool {
	files, err := ioutil.ReadDir(dir)
	return err == nil && len(files) == 0
}

type newOptions struct {
	Template string `help:"Project template to use"`
	List     bool   `help:"List available project templates"`
	Gae      bool   `help:"Create an App Engine hybrid project"`
}

func newCommand(args []string, opts *newOptions) error {
	tmpls, err := getAvailableTemplates()
	if err != nil {
		return err
	}
	if opts.List {
		w := tabwriter.NewWriter(os.Stdout, 8, 4, 2, ' ', 0)
		for _, v := range tmpls {
			fmt.Fprintf(w, "%s:\t%s\n", v.Name, v.Description)
		}
		return w.Flush()
	}
	if len(args) == 0 {
		return errors.New("missing directory name")
	}
	name := args[0]
	for _, v := range tmpls {
		if v.Name == opts.Template {
			if exists, _ := fileutil.Exists(name); exists && !isEmptyDir(name) {
				return fmt.Errorf("%s already exists", name)
			}
			if err := os.MkdirAll(name, 0755); err != nil {
				return err
			}
			r, err := getTemplateReader(v.URL, opts.Gae)
			if err != nil {
				return err
			}
			hdr, err := r.Next()
			if err != nil {
				return err
			}
			// Data to pass to templates
			tmplData := map[string]interface{}{
				"Port":      10000 + rand.Intn(20001), // random port between 10k and 30k
				"AppSecret": stringutil.RandomPrintable(64),
				"DevSecret": stringutil.RandomPrintable(64),
			}
			for hdr != nil {
				p := filepath.Join(name, filepath.FromSlash(hdr.Name))
				info := hdr.FileInfo()
				if info.IsDir() {
					log.Debugf("creating directory %s", p)
					if err := os.MkdirAll(p, info.Mode()); err != nil {
						panic(err)
					}
				} else if filepath.Base(p) != ".keep" {
					// .keep is used to make git keep the directory
					data, err := ioutil.ReadAll(r)
					if err != nil {
						panic(err)
					}
					if ext := filepath.Ext(p); ext == ".gtmpl" {
						p = p[:len(p)-len(ext)]
						tmpl, err := template.New(filepath.Base(p)).Parse(string(data))
						if err != nil {
							panic(err)
						}
						var buf bytes.Buffer
						if err := tmpl.Execute(&buf, tmplData); err != nil {
							panic(err)
						}
						data = buf.Bytes()
					}
					log.Debugf("writing file %s", p)
					if err := ioutil.WriteFile(p, data, info.Mode()); err != nil {
						return err
					}
				}
				hdr, err = r.Next()
				if err != nil && err != io.EOF {
					return err
				}
			}
			return nil
		}
	}
	available := generic.Map(tmpls, func(t *project.Template) string { return t.Name }).([]string)
	return fmt.Errorf("template %s not found, availble ones are: %s", opts.Template, strings.Join(available, ", "))
}
