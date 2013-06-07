package orm

import (
	"errors"
)

var (
	ErrNotFound = errors.New("no results found")
	ErrNoSql    = errors.New("driver is not using database/sql")
)
