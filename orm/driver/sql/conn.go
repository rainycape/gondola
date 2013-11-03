// +build go1.2

package sql

import (
	"database/sql"
)

func setMaxConns(db *sql.DB, n int) {
	db.SetMaxOpenConns(n)
}
