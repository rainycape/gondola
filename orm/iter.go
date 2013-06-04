package orm

type Iter interface {
	Next(out interface{}) bool
	Err() error
}
