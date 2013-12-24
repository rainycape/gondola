package operation

type Operator int

const (
	OpAdd Operator = iota + 1
	OpSub
	OpSet
)

type Field string

type Operation struct {
	Operator Operator
	Field    string
	Value    interface{}
}

func Add(field string, value int) *Operation {
	return &Operation{
		Operator: OpAdd,
		Field:    field,
		Value:    value,
	}
}

func Inc(field string) *Operation {
	return Add(field, 1)
}

func Sub(field string, value int) *Operation {
	return &Operation{
		Operator: OpSub,
		Field:    field,
		Value:    value,
	}
}

func Dec(field string) *Operation {
	return Sub(field, 1)
}

func Set(field string, value interface{}) *Operation {
	return &Operation{
		Operator: OpSet,
		Field:    field,
		Value:    value,
	}
}
