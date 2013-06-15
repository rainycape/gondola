package query

type Q interface {
	// This function exists only to avoid declaring Q
	// as an empty interface. Otherwise, the user might
	// accidentally swap the parameters to Update or Insert
	// and won't get a compiler error.
	q()
}

type Field struct {
	Field string
	Value interface{}
}

func (f Field) q() {
}

type Eq struct {
	Field
}

type Neq struct {
	Field
}

type Lt struct {
	Field
}

type Lte struct {
	Field
}

type Gt struct {
	Field
}

type Gte struct {
	Field
}

type In struct {
	Field
}

type Combinator struct {
	Conditions []Q
}

func (c Combinator) q() {
}

type And struct {
	Combinator
}

type Or struct {
	Combinator
}
