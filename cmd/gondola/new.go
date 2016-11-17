package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"text/template"

	"gnd.la/log"
	"gnd.la/util/fileutil"
	"gnd.la/util/generic"
	"gnd.la/util/stringutil"
	"gnd.la/util/yaml"

	"os/user"

	"archive/zip"
	"net/http"

	"github.com/rainycape/command"
)

const (
	metaFile = "_template.yaml"
)

// Template represents a project template used to initialize
// a new project.
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

// Data returns the data for the template files packed as a .tar.gz
func (t *Template) Data(gae bool) ([]byte, error) {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	tw := tar.NewWriter(zw)
	// Add common files first
	if filepath.Base(t.dir) != "blank" {
		if err := t.addFilesFromSibling(tw, "_common"); err != nil {
			return nil, err
		}
	}
	// Then add gae-specific files, so they may overwrite the _common ones
	if gae {
		if err := t.addFilesFromSibling(tw, "_appengine"); err != nil {
			return nil, err
		}
	}
	// Finally, add project specific files, so they overwrite anything else
	if err := t.addFiles(tw, t.dir); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExpandInto expands the template into the given directory, creating it. If
// the directory already exists and it's non-empty, and error is returned.
func (t *Template) ExpandInto(dir string, gae bool) error {
	if exists, _ := fileutil.Exists(dir); exists && !isEmptyDir(dir) {
		return fmt.Errorf("%s already exists", dir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := t.Data(gae)
	if err != nil {
		return err
	}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer zr.Close()
	r := tar.NewReader(zr)

	hdr, err := r.Next()
	if err != nil {
		return err
	}
	// Data to pass to templates
	tmplData := map[string]interface{}{
		"Port":             10000 + rand.Intn(20001), // random port between 10k and 30k
		"AppSecret":        stringutil.RandomPrintable(64),
		"AppEncryptionKey": stringutil.RandomPrintable(32),
		"DevSecret":        stringutil.RandomPrintable(64),
		"DevEncryptionKey": stringutil.RandomPrintable(32),
	}
	for hdr != nil {
		p := filepath.Join(dir, filepath.FromSlash(hdr.Name))
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

// LoadTemplates returns all the templates found in
// the given directory.
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
		if !v.IsDir() {
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

func isEmptyDir(dir string) bool {
	files, err := ioutil.ReadDir(dir)
	return err == nil && len(files) == 0
}

func updateTemplates(dir string, etagFile string) error {
	const zipBall = "https://github.com/rainycape/gondola-project-templates/archive/master.zip"
	resp, err := http.Get(zipBall)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if etag := resp.Header.Get("Etag"); etag != "" {
		if etagFile != "" {
			etagData, _ := ioutil.ReadFile(etagFile)
			if string(etagData) == etag {
				// Up to date
				return nil
			}
		}
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		// Skip directories, since use .keep files for that
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		// Strip top folder name, since it's the repository name. Note
		// that the zip package only uses forward slashes.
		pos := strings.IndexByte(f.Name, '/')
		name := f.Name[pos+1:]
		dest := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		ff, err := f.Open()
		if err != nil {
			return err
		}
		defer ff.Close()
		w, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer w.Close()
		if _, err := io.Copy(w, ff); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		if err := ff.Close(); err != nil {
			return err
		}
	}

	// Update Etag
	if etag := resp.Header.Get("Etag"); etag != "" {
		ioutil.WriteFile(etagFile, []byte(etag), 0644)
	}

	return nil
}

type newOptions struct {
	Template string `help:"Project template to use"`
	List     bool   `help:"List available project templates"`
	Gae      bool   `help:"Create an App Engine hybrid project"`
}

func newCommand(args *command.Args, opts *newOptions) error {
	var projectTmplDir string
	if envDir := os.Getenv("GONDOLA_PROJECT_TEMPLATES"); envDir != "" {
		projectTmplDir = envDir
	} else {
		// Check for updated templates
		usr, err := user.Current()
		if err != nil {
			return err
		}
		cache := filepath.Join(usr.HomeDir, ".gondola", "project-templates")
		projectTmplDir = filepath.Join(cache, "templates")
		etagFile := filepath.Join(cache, "etag")
		if err := updateTemplates(projectTmplDir, etagFile); err != nil {
			// Check if the directory exists
			if st, _ := os.Stat(projectTmplDir); st == nil || !st.IsDir() {
				return err
			}
		}
	}
	tmpls, err := LoadTemplates(projectTmplDir)
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
	if len(args.Args()) == 0 {
		return errors.New("missing directory name")
	}
	var projectTmpl *Template
	for _, v := range tmpls {
		if v.Name == opts.Template {
			projectTmpl = v
			break
		}
	}
	dir := args.Args()[0]
	if projectTmpl != nil {
		return projectTmpl.ExpandInto(dir, opts.Gae)
	}
	available := generic.Map(tmpls, func(t *Template) string { return t.Name }).([]string)
	return fmt.Errorf("template %s not found, availble ones are: %s", opts.Template, strings.Join(available, ", "))
}
