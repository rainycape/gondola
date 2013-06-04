package sql

import (
	"reflect"
)

type transform struct {
	In  reflect.Value
	Out reflect.Value
}
