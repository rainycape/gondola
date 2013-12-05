package types

import (
	"bytes"
	"fmt"
	"reflect"
)

type Tag struct {
	name   string
	values map[string]string
}

func (t *Tag) Name() string {
	return t.name
}

func (t *Tag) Has(opt string) bool {
	_, ok := t.values[opt]
	return ok
}

func (t *Tag) Value(key string) string {
	return t.values[key]
}

func (t *Tag) IntValue(key string) (int, bool) {
	v := t.Value(key)
	if v != "" {
		var val int
		ret := Parse(t.Value(key), &val)
		return val, ret == nil
	}
	return 0, false
}

// Commonly used tag fields

func (t *Tag) CodecName() string {
	return t.Value("codec")
}

func (t *Tag) PipeName() string {
	return t.Value("pipe")
}

func (t *Tag) Optional() bool {
	return t.Has("optional")
}

func (t *Tag) Required() bool {
	return t.Has("required")
}

func (t *Tag) Alphanumeric() bool {
	return t.Has("alphanumeric")
}

func (t *Tag) MaxLength() (int, bool) {
	return t.IntValue("max_length")
}

func (t *Tag) MinLength() (int, bool) {
	return t.IntValue("min_length")
}

func (t *Tag) IsEmpty() bool {
	return t.name == "" && len(t.values) == 0
}

func splitFields(tag string) (string, map[string]string, error) {
	const (
		stateKey = iota
		stateValue
		stateValueQuoted
		stateEscape
	)
	hasName := false
	var name string
	var key string
	var prevState int
	state := stateKey
	var buf bytes.Buffer
	values := make(map[string]string)
	for ii, v := range []byte(tag) {
		switch v {
		case ',':
			if state != stateValueQuoted {
				if state == stateEscape {
					return "", nil, fmt.Errorf("unknown escape sequence \\%s at %d", string(v), ii)
				}
				if hasName {
					if key != "" {
						values[key] = buf.String()
						key = ""
					} else {
						values[buf.String()] = ""
					}
				} else {
					name = buf.String()
					hasName = true
				}
				buf.Reset()
				state = stateKey
			} else {
				buf.WriteByte(v)
			}
		case '\'':
			if state == stateValue {
				if buf.Len() == 0 {
					state = stateValueQuoted
				} else {
					buf.WriteByte(v)
				}
			} else if state == stateValueQuoted {
				values[key] = buf.String()
				key = ""
				buf.Reset()
				state = stateKey
			} else if state == stateEscape {
				buf.WriteByte(v)
				state = prevState
			} else {
				return "", nil, fmt.Errorf("illegal character ' in key at %d", ii)
			}
		case '\\':
			if state == stateEscape {
				buf.WriteByte(v)
			} else {
				prevState = state
				state = stateEscape
			}
		case '=':
			if state == stateKey {
				key = buf.String()
				buf.Reset()
				state = stateValue
			} else {
				buf.WriteByte(v)
			}
		default:
			if state == stateEscape {
				return "", nil, fmt.Errorf("unknown escape sequence \\%s at %d", string(v), ii)
			}
			buf.WriteByte(v)
		}
	}
	switch state {
	case stateKey:
		if k := buf.String(); k != "" {
			if hasName {
				values[k] = ""
			} else {
				name = k
			}
		}
	case stateValue:
		values[key] = buf.String()
	default:
		return "", nil, fmt.Errorf("unexpected end at %d", len(tag))
	}
	return name, values, nil
}

// ParseTag parses a Gondola style struct tag field from
// the given tag string.
func ParseTag(tag string) (*Tag, error) {
	name, values, err := splitFields(tag)
	if err != nil {
		return nil, err
	}
	return &Tag{name, values}, nil
}

// MustParseTag works like ParseTag, but panics if there's an error.
func MustParseTag(tag string) *Tag {
	t, err := ParseTag(tag)
	if err != nil {
		panic(err)
	}
	return t
}

func NewTag(field reflect.StructField, alternatives []string) *Tag {
	for _, v := range alternatives {
		t := field.Tag.Get(v)
		if t != "" {
			return MustParseTag(t)
		}
	}
	return MustParseTag("")
}

func NewTagNamed(field reflect.StructField, name string) *Tag {
	return MustParseTag(field.Tag.Get(name))
}

func NewStringTagNamed(tag string, name string) *Tag {
	t := reflect.StructTag(tag)
	return MustParseTag(t.Get(name))
}
