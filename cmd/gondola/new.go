package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"gnd.la/admin"
	"gnd.la/app"
	"gnd.la/internal/project"
	"gnd.la/log"
	"gnd.la/net/urlutil"
	"gnd.la/util/fileutil"
	"gnd.la/util/generic"
	"gnd.la/util/stringutil"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"text/template"
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

func NewCmd(ctx *app.Context) {
	tmpls, err := getAvailableTemplates()
	if err != nil {
		panic(err)
	}
	var list bool
	var ptemplate string
	var gae bool
	ctx.ParseParamValue("list", &list)
	ctx.ParseParamValue("template", &ptemplate)
	ctx.ParseParamValue("gae", &gae)
	if list {
		w := tabwriter.NewWriter(os.Stdout, 8, 4, 2, ' ', 0)
		for _, v := range tmpls {
			fmt.Fprintf(w, "%s:\t%s\n", v.Name, v.Description)
		}
		if err := w.Flush(); err != nil {
			panic(err)
		}
	}
	name := ctx.IndexValue(0)
	if name == "" {
		admin.UsageError("missing directory name")
	}
	for _, v := range tmpls {
		if v.Name == ptemplate {
			if exists, _ := fileutil.Exists(name); exists && !isEmptyDir(name) {
				admin.Errorf("%s already exists", name)
			}
			if err := os.MkdirAll(name, 0755); err != nil {
				panic(err)
			}
			r, err := getTemplateReader(v.URL, gae)
			if err != nil {
				panic(err)
			}
			hdr, err := r.Next()
			if err != nil {
				panic(err)
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
						panic(err)
					}
				}
				hdr, err = r.Next()
				if err != nil && err != io.EOF {
					panic(err)
				}
			}
			return
		}
	}
	available := generic.Map(tmpls, func(t *project.Template) string { return t.Name }).([]string)
	admin.Errorf("template %s not found, availble ones are: %s", ptemplate, strings.Join(available, ", "))
}

func init() {
	admin.Register(NewCmd, &admin.Options{
		Name:  "New",
		Help:  "Create a new Gondola project",
		Usage: "<dir_name>",
		Flags: admin.Flags(
			admin.StringFlag("template", "hello", "Project template to use"),
			admin.BoolFlag("list", false, "List available project templates"),
			admin.BoolFlag("gae", false, "Create an App Engine hybrid project"),
		),
	})
}
