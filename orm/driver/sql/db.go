package sql

import (
	"database/sql"
	"errors"
	"strings"
)

var (
	ErrNoRows           = sql.ErrNoRows
	ErrFuncNotSupported = errors.New("function not supported")
)

type Queryier interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type Executor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type DB interface {
	Executor
	Queryier
	QuoteString(s string) string
	QuoteIdentifier(s string) string
}

type queryExecutor interface {
	Queryier
	Executor
}

type db struct {
	sqlDb  *sql.DB
	tx     *sql.Tx
	db     queryExecutor
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

func (d *db) QuoteString(s string) string {
	return d.quoteWith(s, d.driver.backend.StringQuote())
}

func (d *db) QuoteIdentifier(s string) string {
	return d.quoteWith(s, d.driver.backend.IdentifierQuote())
}

func (d *db) quoteWith(s string, q byte) string {
	qu := string(q)
	var escaped string
	if q == '\'' {
		escaped = strings.Replace(s, "'", "''", -1)
	} else {
		escaped = strings.Replace(s, qu, "\\"+qu, -1)
	}
	return qu + escaped + qu
}
