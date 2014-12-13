package orm

import (
	"gnd.la/orm/driver"
)

var (
	ErrInTransaction    = driver.ErrInTransaction
	ErrNotInTransaction = driver.ErrNotInTransaction
	ErrFinished         = driver.ErrFinished
)

type Tx struct {
	Orm
	// Parent orm
	o    *Orm
	tx   driver.Tx
	done bool
}

// Begin just returns ErrInTransaction when called
// from a transaction.
func (t *Tx) Begin() (*Tx, error) {
	return nil, ErrInTransaction
}

// Commit commits the current transaction. If the transaction
// was already committed or rolled back, it returns ErrFinished.
func (t *Tx) Commit() error {
	if t.done {
		return ErrFinished
	}
	if t.logger != nil {
		t.logger.Debug("Commiting transaction")
	}
	err := t.tx.Commit()
	if err != nil {
		return err
	}
	t.done = true
	return nil
}

// MustCommit works like Commit, but panics if there's an error.
func (t *Tx) MustCommit() {
	if err := t.Commit(); err != nil {
		panic(err)
	}
}

// Rollback rolls back the current transaction. If the transaction
// was already committed or rolled back, it returns ErrFinished.
func (t *Tx) Rollback() error {
	if t.done {
		return ErrFinished
	}
	if t.logger != nil {
		t.logger.Debug("Rolling back transaction")
	}
	err := t.tx.Rollback()
	if err != nil {
		return err
	}
	t.done = true
	return nil
}

// MustRollback works like Rollback, but panics if there's an error.
func (t *Tx) MustRollback() {
	if err := t.Rollback(); err != nil {
		panic(err)
	}
}

// Close finishes this transaction with a rollback if it hasn't been
// commited or rolled back yet. It's intended to be called using defer
// so you can cleanly release a transaction even if your code
// panics before commiting it. e.g.
//
//  tx, err := o.Begin()
//  if err != nil {
//	panic(err)
//  }
//  defer tx.Close()
//  // do some stuff with tx
//  tx.MustCommit()
func (t *Tx) Close() {
	if !t.done {
		t.Rollback()
	}
}

func (t *Tx) compileTimeInterfaceTest() Interface {
	return t
}
