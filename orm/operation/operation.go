package operation

type Operator int

const (
	Add Operator = iota + 1
	Sub
	Set
)

type Field string

type Operation struct {
	Operator Operator
	Field    string
	Value    interface{}
}

func Inc(field string) *Operation {
	return IncBy(field, 1)
}

func IncBy(field string, value int) *Operation {
	return &Operation{
		Operator: Add,
		Field:    field,
		Value:    value,
	}
}

func Dec(field string) *Operation {
	return DecBy(field, 1)
}

func DecBy(field string, value int) *Operation {
	return &Operation{
		Operator: Sub,
		Field:    field,
		Value:    value,
	}
}
