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

type Multiple struct {
	Conditions []Q
}

type And struct {
	Multiple
}

type Or struct {
	Multiple
}
