package driver

import (
	"errors"
)

var (
	ErrInTransaction    = errors.New("already in transaction")
	ErrNotInTransaction = errors.New("not in transaction")
	ErrFinished         = errors.New("transaction was already finished")
)

type Tx interface {
	Conn
	Commit() error
	Rollback() error
}
