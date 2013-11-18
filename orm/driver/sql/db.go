package sql

import (
	"database/sql"
)

type DB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type db struct {
	sqlDb  *sql.DB
	tx     *sql.Tx
	db     DB
	driver *Driver
}

func (d *db) Exec(query string, args ...interface{}) (sql.Result, error) {
	d.driver.debugq(query, args)
	return d.db.Exec(query, args...)
}

func (d *db) Query(query string, args ...interface{}) (*sql.Rows, error) {
	d.driver.debugq(query, args)
	return d.db.Query(query, args...)
}

func (d *db) QueryRow(query string, args ...interface{}) *sql.Row {
	d.driver.debugq(query, args)
	return d.db.QueryRow(query, args...)
}
