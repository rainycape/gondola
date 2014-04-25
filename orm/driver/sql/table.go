package sql

import (
	"strings"
)

type ConstraintType int

const (
	ConstraintNotNull = 1 + iota
	ConstraintUnique
	ConstraintPrimaryKey
	ConstraintForeignKey
)

type FieldOption int

const (
	OptionAutoIncrement = 1 + iota
)

type Reference string

const refSep = "|"

func (r Reference) Table() string {
	s := string(r)
	pos := strings.Index(s, refSep)
	return s[:pos]
}

func (r Reference) Field() string {
	s := string(r)
	pos := strings.Index(s, refSep)
	return s[pos+1:]
}

func MakeReference(table string, field string) Reference {
	return Reference(table + refSep + field)
}

type Field struct {
	Name        string
	Type        string
	Default     string
	Options     []FieldOption
	Constraints []*Constraint
}

func (f *Field) AddOption(opt FieldOption) {
	f.Options = append(f.Options, opt)
}

func (f *Field) HasOption(opt FieldOption) bool {
	for _, v := range f.Options {
		if v == opt {
			return true
		}
	}
	return false
}

func (f *Field) AddConstraint(ct ConstraintType) {
	f.Constraints = append(f.Constraints, &Constraint{Type: ct})
}

func (f *Field) Constraint(ct ConstraintType) *Constraint {
	for _, v := range f.Constraints {
		if v.Type == ct {
			return v
		}
	}
	return nil
}

type Constraint struct {
	Type       ConstraintType
	References Reference
}

type Table struct {
	Fields      []*Field
	Constraints []*Constraint
}

func (t *Table) PrimaryKeys() []string {
	var keys []string
	for _, v := range t.Fields {
		if v.Constraint(ConstraintPrimaryKey) != nil {
			keys = append(keys, v.Name)
		}
	}
	return keys
}
