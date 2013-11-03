package sql

type scanner interface {
	Scan(src interface{}) error
	IsNil() bool
	Put()
}
