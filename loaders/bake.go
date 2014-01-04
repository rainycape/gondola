package loaders

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"gnd.la/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func writeBytes(buf *bytes.Buffer, data []byte, raw bool) {
	if raw {
		buf.WriteString(fmt.Sprintf("loaders.RawString(%q)", string(data)))
	} else {
		buf.WriteString("[]byte{")
		for ii, v := range data {
			s := fmt.Sprintf("%q,", v)
			if len(s) > 3 {
				if s[2] == 'u' {
					s = "'\\x" + s[5:]
				} else if s == "'\\x00'," {
					s = "0,"
				}
			}
			buf.WriteString(s)
			if ii%16 == 0 && ii != len(data)-1 {
				buf.WriteByte('\n')
			}
		}
		buf.Truncate(buf.Len() - 1)
		buf.WriteString("}")
	}
}

func zipCompress(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for k, v := range files {
		f, err := w.Create(k)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(v); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func tgzCompress(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	z, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	w := tar.NewWriter(z)
	for k, v := range files {
		hdr := &tar.Header{
			Name: k,
			Size: int64(len(v)),
		}
		if err := w.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := w.Write(v); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type BakeCompress int

const (
	CompressNone BakeCompress = iota
	CompressFlate
	CompressZip
	CompressTgz
)

func Bake(buf *bytes.Buffer, dir string, extensions []string, compress BakeCompress) error {
	var exts map[string]bool
	if len(extensions) > 0 {
		exts := make(map[string]bool)
		for _, v := range extensions {
			if v == "" {
				continue
			}
			if v[0] != '.' {
				v = "." + v
			}
			exts[strings.ToLower(v)] = true
		}
	}
	files := make(map[string][]byte)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			if exts == nil || exts[strings.ToLower(filepath.Ext(path))] {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					return fmt.Errorf("error reading %s: %s", path, err)
				}
				rel := path[len(dir):]
				if rel[0] == '/' {
					rel = rel[1:]
				}
				files[rel] = contents
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	switch compress {
	case CompressNone:
		buf.WriteString("loaders.MapLoader(map[string][]byte{\n")
		for k, v := range files {
			fmt.Fprintf(buf, "%q:", k)
			writeBytes(buf, v, false)
			buf.WriteString(",\n")
		}
		buf.WriteString("})\n")
	case CompressFlate:
		buf.WriteString("loaders.FlateLoader(loaders.MapLoader(map[string][]byte{\n")
		for k, v := range files {
			fmt.Fprintf(buf, "%q:", k)
			cmp, err := flateCompress(v)
			if err != nil {
				return fmt.Errorf("error compressing %s with flate: %s", k, err)
			}
			writeBytes(buf, cmp, false)
			buf.WriteString(",\n")
		}
		buf.WriteString("}))\n")
	case CompressZip:
		buf.WriteString("loaders.ZipLoader(")
		data, err := zipCompress(files)
		if err != nil {
			return fmt.Errorf("error compressing zip: %s", err)
		}
		log.Debugf("Compressed with zip to %d bytes", len(data))
		writeBytes(buf, data, true)
		buf.WriteString(")\n")
	case CompressTgz:
		buf.WriteString("loaders.TgzLoader(")
		data, err := tgzCompress(files)
		if err != nil {
			return fmt.Errorf("error compressing tgz: %s", err)
		}
		log.Debugf("Compressed with tgz to %d bytes", len(data))
		writeBytes(buf, data, true)
		buf.WriteString(")\n")
	default:
		return fmt.Errorf("invalid compression method %d", compress)
	}
	return nil
}
