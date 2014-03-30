package orm

import (
	"errors"
)

var (
	// ErrNotSql indicates that the current driver is not using database/sql.
	ErrNoSql = errors.New("driver is not using database/sql")
)
