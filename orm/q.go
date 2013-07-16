package orm

import (
	"gondola/orm/query"
)

func Eq(field string, value interface{}) query.Q {
	return &query.Eq{
		Field: query.Field{
			Field: field,
			Value: value,
		},
	}
}

func Neq(field string, value interface{}) query.Q {
	return &query.Neq{
		Field: query.Field{
			Field: field,
			Value: value,
		},
	}
}

func Lt(field string, value interface{}) query.Q {
	return &query.Lt{
		Field: query.Field{
			Field: field,
			Value: value,
		},
	}
}

func Lte(field string, value interface{}) query.Q {
	return &query.Lte{
		Field: query.Field{
			Field: field,
			Value: value,
		},
	}
}

func Gt(field string, value interface{}) query.Q {
	return &query.Gt{
		Field: query.Field{
			Field: field,
			Value: value,
		},
	}
}

func Gte(field string, value interface{}) query.Q {
	return &query.Gte{
		Field: query.Field{
			Field: field,
			Value: value,
		},
	}
}

func In(field string, value interface{}) query.Q {
	return &query.In{
		Field: query.Field{
			Field: field,
			Value: value,
		},
	}
}

func And(qs ...query.Q) query.Q {
	return &query.And{
		Combinator: query.Combinator{
			Conditions: qs,
		},
	}
}

func Or(qs ...query.Q) query.Q {
	return &query.Or{
		Combinator: query.Combinator{
			Conditions: qs,
		},
	}
}

// These are shorthand forms for the previous

// Between is equivalent to field > begin AND field < end.
func Between(field string, begin interface{}, end interface{}) query.Q {
	return And(Gt(field, begin), Lt(field, end))
}

// CBetween stands for closed between and is equivalent to field >= begin AND field <= end.
func CBetween(field string, begin interface{}, end interface{}) query.Q {
	return And(Gte(field, begin), Lte(field, end))
}

// LCBetween stands for left closed between and is equivalent to field >= begin AND field < end.
func LCBetween(field string, begin interface{}, end interface{}) query.Q {
	return And(Gte(field, begin), Lt(field, end))
}

// RCBetween stands for right closed between and is equivalent to field > begin AND field <= end.
func RCBetween(field string, begin interface{}, end interface{}) query.Q {
	return And(Gt(field, begin), Lte(field, end))
}
