package astutil

import (
	"go/ast"
	"go/token"
	"gondola/pkg"
	"gondola/types"
)

type String struct {
	Value    string
	Position *token.Position
}

// TODO: Allow context and plural to be specified with |

func (s *String) Singular() string {
	return s.Value
}

func (s *String) Plural() string {
	return ""
}

func (s *String) Context() string {
	return ""
}

// Strings returns a list of string declarations of the given type
// (as a qualified name).
func Strings(fset *token.FileSet, f *ast.File, typ string) ([]*String, error) {
	pkg, tname := pkg.SplitQualifiedName(typ)
	pname, ok := Imports(f, pkg)
	if !ok {
		// Not imported
		return nil, nil
	}
	var strings []*String
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ValueSpec:
			p, t := Selector(x.Type)
			if p == pname && t == tname {
				for _, v := range x.Values {
					if s, pos := StringLiteral(fset, v); s != "" && pos != nil {
						strings = append(strings, &String{s, pos})
					}
				}
			}
		}
		return true
	})
	return strings, nil
}

func TagFields(fset *token.FileSet, f *ast.File, tagField string) ([]*String, error) {
	var strings []*String
	var err error
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Field:
			if x.Tag != nil {
				if s, pos := StringLiteral(fset, x.Tag); s != "" && pos != nil {
					for _, v := range tagKeys(s) {
						t := types.NewStringTagNamed(s, v)
						if val := t.Value(tagField); val != "" {
							strings = append(strings, &String{val, pos})
						}
					}
				}
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return strings, nil
}

func tagKeys(tag string) []string {
	var keys []string
	s := 0
	for ii := 0; ii < len(tag); ii++ {
		if tag[ii] == ' ' {
			s = ii + 1
			continue
		}
		if tag[ii] == ':' {
			keys = append(keys, tag[s:ii])
		}
		if tag[ii] == '"' {
			ii++
			for tag[ii] != '"' {
				ii++
			}
		}
	}
	return keys
}