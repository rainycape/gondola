package driver

type Iter interface {
	Next(out ...interface{}) bool
	Err() error
	Close() error
}
