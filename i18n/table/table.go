package table

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"gnd.la/encoding/binary"
)

type Formula func(n int) int

type Table struct {
	translations map[string]Translation
	formula      Formula
}

func (t *Table) Singular(ctx string, msg string) string {
	k := Key(ctx, msg, "")
	if tr := t.translations[k]; len(tr) > 0 {
		return tr[0]
	}
	return msg
}

func (t *Table) Plural(ctx string, singular string, plural string, n int) string {
	k := Key(ctx, singular, plural)
	if tr := t.translations[k]; tr != nil {
		ii := t.formula(n)
		if ii < len(tr) {
			return tr[ii]
		}
	}
	if n == 1 {
		return singular
	}
	return plural
}

func (t *Table) Encode() (string, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return "", err
	}
	if err := binary.Write(w, binary.BigEndian, int32(len(t.translations))); err != nil {
		return "", err
	}
	for k, v := range t.translations {
		if err := writeString(w, k); err != nil {
			return "", err
		}
		if err := binary.Write(w, binary.BigEndian, int32(len(v))); err != nil {
			return "", err
		}
		for _, s := range v {
			if err := writeString(w, s); err != nil {
				return "", err
			}
		}
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (t *Table) Update(other *Table) error {
	if other.formula != nil {
		t.formula = other.formula
	}
	for k, v := range other.translations {
		t.translations[k] = v
	}
	return nil
}

func Decode(data string) (*Table, error) {
	r, err := gzip.NewReader(strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var count int32
	if err := binary.Read(r, binary.BigEndian, &count); err != nil {
		return nil, err
	}
	c := int(count)
	translations := make(map[string]Translation, c)
	for ii := 0; ii < c; ii++ {
		key, err := readString(r)
		if err != nil {
			return nil, err
		}
		var tcount int32
		if err := binary.Read(r, binary.BigEndian, &tcount); err != nil {
			return nil, err
		}
		tc := int(tcount)
		value := make([]string, tc)
		for jj := 0; jj < tc; jj++ {
			tr, err := readString(r)
			if err != nil {
				return nil, err
			}
			value[jj] = tr
		}
		translations[key] = value
	}
	return &Table{
		translations: translations,
	}, nil
}

func New(formula Formula, translations map[string]Translation) (*Table, error) {
	return &Table{
		translations: translations,
		formula:      formula,
	}, nil
}

func writeString(w io.Writer, s string) error {
	b := []byte(s)
	if err := binary.Write(w, binary.BigEndian, int32(len(b))); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

func readString(r io.Reader) (string, error) {
	var s int32
	if err := binary.Read(r, binary.BigEndian, &s); err != nil {
		return "", err
	}
	b := make([]byte, int(s))
	if _, err := r.Read(b); err != nil {
		return "", err
	}
	return string(b), nil
}
