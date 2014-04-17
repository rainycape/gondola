// +build appengine

package datastore

import (
	"errors"

	"appengine/datastore"
)

var (
	errNotInserted             = errors.New("no rows where inserted")
	errJoinNotSupported        = errors.New("datastore driver does not support JOIN")
	errTransactionNotSupported = errors.New("datastore driver does not support transactions")
)

type result struct {
	key   *datastore.Key
	count int
}

func (r *result) LastInsertId() (int64, error) {
	if r.key != nil {
		return r.key.IntID(), nil
	}
	return 0, errNotInserted
}

func (r *result) RowsAffected() (int64, error) {
	return int64(r.count), nil
}
