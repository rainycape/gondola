package query

type Q interface {
}

type Field struct {
	Field string
	Value interface{}
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

type And struct {
	Combinator
}

type Or struct {
	Combinator
}
