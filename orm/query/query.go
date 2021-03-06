package query

import (
	"fmt"
	"strings"
)

type Q interface {
	// Qualified Go name. It might reference a
	// type like type|field
	FieldName() string
	SubQ() []Q
}

// F represents a reference to a field. This is used to disambiguate
// when the value in a Q refers to a string or a field.
type F string

func (f F) String() string {
	return fmt.Sprintf("F(%s)", string(f))
}

// Type Subquery represents a literal subquery passed as a Q condition.
// It can be used with Eq, Neq, In, etc...
type Subquery string

type Field struct {
	Field string
	Value interface{}
}

func (f *Field) FieldName() string {
	return f.Field
}

func (f *Field) SubQ() []Q {
	return nil
}

type Eq struct {
	Field
}

func (e *Eq) String() string {
	return qDesc(&e.Field, "= ")
}

type Neq struct {
	Field
}

func (n *Neq) String() string {
	return qDesc(&n.Field, "!= ")
}

type Contains struct {
	Field
}

func (c *Contains) String() string {
	return qDesc(&c.Field, "CONTAINS (") + ")"
}

type Lt struct {
	Field
}

func (l *Lt) String() string {
	return qDesc(&l.Field, "< ")
}

type Lte struct {
	Field
}

func (l *Lte) String() string {
	return qDesc(&l.Field, "<= ")
}

type Gt struct {
	Field
}

func (g *Gt) String() string {
	return qDesc(&g.Field, "> ")
}

type Gte struct {
	Field
}

func (g *Gte) String() string {
	return qDesc(&g.Field, ">= ")
}

type In struct {
	Field
}

type Combinator struct {
	Conditions []Q
}

func (c *Combinator) FieldName() string {
	return ""
}

func (c *Combinator) SubQ() []Q {
	return c.Conditions
}

type And struct {
	Combinator
}

func (a *And) String() string {
	return combDesc(&a.Combinator, "AND")
}

type Or struct {
	Combinator
}

func (o *Or) String() string {
	return combDesc(&o.Combinator, "OR")
}

type Join struct {
	Model interface{}
	Field string
	Query Q
}

// Operator represents an arbitrary operator which is passed
// as-is to the underlying database. It conforms to the
// Q interface.
type Operator struct {
	Field
	// The operator itself.
	Operator string
}

func (o *Operator) String() string {
	return qDesc(&o.Field, o.Operator+" ")
}

func combDesc(c *Combinator, w string) string {
	qs := make([]string, len(c.Conditions))
	for ii, v := range c.Conditions {
		qs[ii] = fmt.Sprintf("%v", v)
	}
	return "(" + strings.Join(qs, " "+w+" ") + ")"
}

func qDesc(f *Field, symb string) string {
	if s, ok := f.Value.(string); ok {
		return fmt.Sprintf("%q %s%q", f.Field, symb, s)
	}
	return fmt.Sprintf("%q %s%v", f.Field, symb, f.Value)
}
