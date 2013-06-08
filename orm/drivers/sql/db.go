package sql

import (
	"database/sql"
)

type DB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type db struct {
	*sql.DB
	driver *Driver
}

func (d *db) Exec(query string, args ...interface{}) (sql.Result, error) {
	d.driver.debugq(query, args)
	return d.DB.Exec(query, args...)
}

func (d *db) QueryRow(query string, args ...interface{}) *sql.Row {
	d.driver.debugq(query, args)
	return d.DB.QueryRow(query, args...)
}
