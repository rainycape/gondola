// Package article includes common types and functions for the articles app and command.
package article

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gnd.la/form/input"
	"gnd.la/util/stringutil"

	"gopkgs.com/vfs.v1"
)

var (
	propertyRe        = regexp.MustCompile(`(?m:^\[(\w+)]\s?=\s?(.*)$)`)
	propertySeparator = []byte("\n\n")

	idKey       = "id"
	titleKey    = "title"
	slugKey     = "slug"
	synopsisKey = "synopsis"
	updatedKey  = "updated"
	priorityKey = "priority"

	timeFormats = []string{
		"2006-01-02",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		time.RFC822,
		time.RFC822Z,
	}
)

const (
	maxSlugLength = 50
)

// Article represents a loaded article.
type Article struct {
	// Id is the article identifier, used by the template
	// function reverse_article.
	Id string
	// Titles includes the current title and then all the previous ones.
	// The previous ones are kept for redirecting old URLs to the
	// new one.
	Titles []string
	// Slugs contains the current slug and all the previous ones (for
	// redirecting all URLs). If no slugs are present, the current title
	Slugs []string
	// is slugified.
	// The article synopsys shown in the articles list. Might be empty.
	Synopsis string
	// A list of times when the article was created.
	Updated []time.Time
	// The priority to sort articles in the index. Articles with the same
	// priority are sorted by title.
	Priority int
	// Text contains the article text, with any properties stripped.
	Text []byte
	// Properties contains unknown properties, to allow forward
	// compatibility. It might be nil.
	Properties map[string][]string
	// Filename indicates the base filename where this article was loaded
	// from, always using UNIX style directory separators. Note that it might be empty.
	Filename string
}

// Title returns the current article title.
func (a *Article) Title() string {
	if len(a.Titles) > 0 {
		return a.Titles[0]
	}
	return ""
}

// Slug returns the current article slug.
func (a *Article) Slug() string {
	if len(a.Slugs) > 0 {
		return a.Slugs[0]
	}
	return stringutil.SlugN(a.Title(), maxSlugLength)
}

// AllSlugs returns the current and all the previous article
// slugs, used for redirects.
func (a *Article) AllSlugs() []string {
	slugs := make([]string, 0, len(a.Slugs)+len(a.Titles))
	slugs = append(slugs, a.Slugs...)
	for _, v := range a.Titles {
		slugs = append(slugs, stringutil.SlugN(v, maxSlugLength))
	}
	return slugs
}

// Created returns the article creation time, or the zero time.Time
// if there are no recorded updates to the article.
func (a *Article) Created() time.Time {
	if len(a.Updated) == 0 {
		return time.Time{}
	}
	t := time.Unix(int64((^uint32(0) >> 1)), 0)
	for _, v := range a.Updated {
		if v.Sub(t) < 0 {
			t = v
		}
	}
	return t
}

// LastUpdate returns the most recent update time, or the zero time.Time
// if there are no recorded updates to the article.
func (a *Article) LastUpdate() time.Time {
	var t time.Time
	for _, v := range a.Updated {
		if v.Sub(t) > 0 {
			t = v
		}
	}
	return t
}

// Set sets a property in the article. For slice properties, it prepends the
// new value to the existing ones.
func (a *Article) Set(key string, value string) error {
	return a.add(key, value, false)
}

func (a *Article) add(key string, value string, begin bool) error {
	var err error
	switch strings.ToLower(key) {
	case idKey:
		err = input.Parse(value, &a.Id)
	case titleKey:
		if begin {
			a.Titles = append([]string{value}, a.Titles...)
		} else {
			a.Titles = append(a.Titles, value)
		}
	case slugKey:
		if begin {
			a.Slugs = append([]string{value}, a.Slugs...)
		} else {
			a.Slugs = append(a.Slugs, value)
		}
	case synopsisKey:
		a.Synopsis = value
	case updatedKey:
		var t time.Time
		switch strings.ToLower(value) {
		case "now":
			t = time.Now().UTC()
		case "today":
			t = time.Now().UTC().Truncate(24 * time.Hour)
		default:
			for _, v := range timeFormats {
				t, err = time.Parse(v, value)
				if err == nil {
					break
				}
			}
		}
		if err == nil {
			if begin {
				a.Updated = append([]time.Time{t}, a.Updated...)
			} else {
				a.Updated = append(a.Updated, t)
			}
		}
	case priorityKey:
		err = input.Parse(value, &a.Priority)
	default:
		if a.Properties == nil {
			a.Properties = make(map[string][]string)
		}
		a.Properties[key] = append(a.Properties[key], value)
	}
	return err
}

func (a *Article) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	buf.Write(a.Text)
	buf.WriteByte('\n')
	buf.Write(propertySeparator)
	if a.Id != "" {
		a.writeProperty(&buf, idKey, a.Id)
	}
	for _, v := range a.Titles {
		a.writeProperty(&buf, titleKey, v)
	}
	for _, v := range a.Slugs {
		a.writeProperty(&buf, slugKey, v)
	}
	if a.Synopsis != "" {
		a.writeProperty(&buf, synopsisKey, a.Synopsis)
	}
	for _, v := range a.Updated {
		a.writeProperty(&buf, updatedKey, v)
	}
	if a.Priority != 0 {
		a.writeProperty(&buf, priorityKey, a.Priority)
	}
	for k, v := range a.Properties {
		for _, prop := range v {
			a.writeProperty(&buf, k, prop)
		}
	}
	n, err := w.Write(buf.Bytes())
	return int64(n), err
}

func (a *Article) writeProperty(buf *bytes.Buffer, key string, value interface{}) {
	var s string
	switch x := value.(type) {
	case []byte:
		s = string(x)
	case string:
		s = x
	case int:
		s = strconv.Itoa(x)
	case time.Time:
		var format string
		if x.Location() != nil && x.Location() != time.UTC {
			format = time.RFC822Z
		} else {
			if x.Second() == 0 {
				if x.Hour() == 0 && x.Minute() == 0 {
					format = "2006-01-02"
				} else {
					format = "2006-01-02 15:04"
				}
			} else {
				format = "2006-01-02 15:04:05"
			}
		}
		s = x.Format(format)
	default:
		panic(fmt.Errorf("can't write property %q %v of type %T", key, value, value))
	}
	if strings.Contains(s, "\n") {
		s = strconv.Quote(s)
	}
	fmt.Fprintf(buf, "[%s] = %s\n", key, s)
}

// New returns a new Article by decoding it from the given io.Reader.
func New(r io.Reader) (*Article, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	article := &Article{}
	var text, props []byte
	// Find last empty line
	sep := bytes.LastIndex(data, propertySeparator)
	if sep >= 0 {
		text = data[:sep]
		props = data[sep:]
	} else {
		text = data
	}
	for _, v := range propertyRe.FindAllSubmatch(props, -1) {
		key := string(v[1])
		value := string(v[2])
		if value != "" && value[0] == '"' && value[len(value)-1] == '"' {
			val, err := strconv.Unquote(value)
			if err != nil {
				return nil, fmt.Errorf("error unquoting %q: %s", value, err)
			}
			value = val
		}
		if err := article.add(key, value, false); err != nil {
			return nil, fmt.Errorf("error setting key %q with value %q: %s", key, string(value), err)
		}
	}
	text = append(text, propertyRe.ReplaceAll(props, nil)...)
	article.Text = bytes.TrimSpace(text)
	return article, nil
}

// Load returns a new Article loading it from the given filename in the given fs.
func Load(fs vfs.VFS, filename string) (*Article, error) {
	f, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	article, err := New(f)
	if err != nil {
		return nil, err
	}
	article.Filename = filename
	return article, nil
}
