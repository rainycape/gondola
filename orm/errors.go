package orm

import (
	"errors"
)

var (
	// ErrNotFound is returned from One() when there are no results.
	ErrNotFound = errors.New("no results found")
	// ErrNotSql indicates that the current driver is not using database/sql.
	ErrNoSql = errors.New("driver is not using database/sql")
)
