package sql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gnd.la/orm/driver"
	"gnd.la/util/generic"
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

func (f *Field) HasConstraint(ct ConstraintType) bool {
	return f.Constraint(ct) != nil
}

func (f *Field) SQL(db *DB, m driver.Model, table *Table) (string, []string, error) {
	return db.Backend().DefineField(db, m, table, f)
}

func (f *Field) Copy() *Field {
	fc := *f
	fc.Options = make([]FieldOption, len(f.Options))
	copy(fc.Options, f.Options)
	fc.Constraints = make([]*Constraint, len(f.Constraints))
	copy(fc.Constraints, f.Constraints)
	return &fc
}

type Constraint struct {
	Type       ConstraintType
	References Reference
}

func (c *Constraint) String() string {
	switch c.Type {
	case ConstraintNotNull:
		return "NOT_NULL"
	case ConstraintUnique:
		return "UNIQUE"
	case ConstraintPrimaryKey:
		return "PRIMARY_KEY"
	case ConstraintForeignKey:
		return fmt.Sprintf("FOREIGN_KEY %s", string(c.References))
	}
	return fmt.Sprintf("unknown constraint type %d", int(c.Type))
}

type Table struct {
	Fields      []*Field
	Constraints []*Constraint
}

func (t *Table) PrimaryKeys() []string {
	var keys []string
	for _, v := range t.Fields {
		if v.HasConstraint(ConstraintPrimaryKey) {
			keys = append(keys, v.Name)
		}
	}
	return keys
}

func (t *Table) definePks(db *DB, m driver.Model) (string, error) {
	pks := t.PrimaryKeys()
	if len(pks) < 2 {
		return "", nil
	}
	pkFields := generic.Map(pks, db.QuoteIdentifier).([]string)
	return fmt.Sprintf("PRIMARY KEY(%s)", strings.Join(pkFields, ", ")), nil
}

func (t *Table) SQL(db *DB, b Backend, m driver.Model, name string) (string, error) {
	var lines []string
	var constraints []string
	for _, v := range t.Fields {
		def, cons, err := v.SQL(db, m, t)
		if err != nil {
			return "", err
		}
		lines = append(lines, def)
		constraints = append(constraints, cons...)
	}
	pk, err := t.definePks(db, m)
	if err != nil {
		return "", err
	}
	if pk != "" {
		lines = append(lines, pk)
	}
	lines = append(lines, constraints...)
	if name == "" {
		name = m.Table()
	}
	// Use IF NOT EXISTS, since the DB user might not have
	// the privileges to inspect the database but still
	// be allowed to read and write from the tables (e.g.
	// Postgres only allows superusers and the owner to
	// inspect the database).
	sql := fmt.Sprintf("\nCREATE TABLE IF NOT EXISTS %s (\n\t%s\n)", db.QuoteIdentifier(name), strings.Join(lines, ",\n\t"))
	return sql, nil
}

type Kind int

const (
	KindInvalid = iota
	KindInteger
	KindFloat
	KindDecimal
	KindChar
	KindVarchar
	KindText
	KindBlob
	KindTime
)

var (
	lengthRe = regexp.MustCompile(`\((\d+)\)`)
)

func fieldLength(typ string) int {
	m := lengthRe.FindStringSubmatch(typ)
	if len(m) > 1 {
		val, _ := strconv.Atoi(m[1])
		return val
	}
	return 0
}

func TypeKind(typ string) (Kind, int) {
	t := strings.ToUpper(typ)
	switch {
	case strings.Contains(t, "INT") || strings.Contains(t, "SERIAL"):
		return KindInteger, 0
	case strings.HasPrefix(t, "VARCHAR") || strings.HasPrefix(t, "CHARACTER VARYING"):
		return KindVarchar, fieldLength(t)
	case strings.HasPrefix(t, "CHAR"):
		return KindChar, fieldLength(t)
	case strings.HasPrefix(t, "BLOB"):
		return KindBlob, 0
	case strings.HasPrefix(t, "TEXT"):
		return KindText, 0
	case t == "DATETIME" || strings.Contains(t, "TIMESTAMP"):
		return KindTime, 0
	}
	return KindInvalid, 0
}
