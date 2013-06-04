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

func And(qs ...query.Q) query.Q {
	return query.And{
		Combinator: query.Combinator{
			Conditions: qs,
		},
	}
}

func Or(qs ...query.Q) query.Q {
	return query.Or{
		Combinator: query.Combinator{
			Conditions: qs,
		},
	}
}
