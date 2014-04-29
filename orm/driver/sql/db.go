package sql

import (
	"bytes"
	"database/sql"
	"errors"
	"strings"

	"gnd.la/orm/driver"
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

type queryExecutor interface {
	Queryier
	Executor
}

type DB struct {
	// database/sql.DB
	sqlDb *sql.DB
	// non-nil only when in transaction
	tx     *sql.Tx
	txDone bool
	// might be eithr sqlDb or tx, depending on
	// if we're inside a transaction or not.
	conn                 queryExecutor
	driver               *Driver
	replacesPlaceholders bool
}

func (d *DB) replacePlaceholders(query string) string {
	var buf bytes.Buffer
	var inQuote, inDoubleQuote bool
	p := 0
	placeholder := d.driver.backend.Placeholder
	written := 0
	last := len(query) - 1
	for ii, ch := range query {
		switch ch {
		case '\'':
			if ii == last || query[ii+1] != '\'' {
				inQuote = !inQuote
			}
		case '"':
			inDoubleQuote = !inDoubleQuote
		case '?':
			if !inQuote && !inDoubleQuote {
				buf.WriteString(query[written:ii])
				buf.WriteString(placeholder(p))
				p++
				written = ii + 1
			}
		}
	}
	if written == 0 {
		return query
	}
	buf.WriteString(query[written:])
	return buf.String()
}

func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	if d.replacesPlaceholders {
		query = d.replacePlaceholders(query)
	}
	d.driver.debugq(query, args)
	return d.conn.Exec(query, args...)
}

func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if d.replacesPlaceholders {
		query = d.replacePlaceholders(query)
	}
	d.driver.debugq(query, args)
	return d.conn.Query(query, args...)
}

func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	if d.replacesPlaceholders {
		query = d.replacePlaceholders(query)
	}
	query = d.replacePlaceholders(query)
	d.driver.debugq(query, args)
	return d.conn.QueryRow(query, args...)
}

func (d *DB) Begin() (*DB, error) {
	if d.tx != nil {
		return nil, driver.ErrInTransaction
	}
	tx, err := d.sqlDb.Begin()
	if err != nil {
		return nil, err
	}
	dc := *d
	dc.tx = tx
	dc.conn = tx
	return &dc, nil
}

func (d *DB) Commit() error {
	if d.tx == nil {
		return driver.ErrNotInTransaction
	}
	d.txDone = true
	return d.tx.Commit()
}

func (d *DB) Rollback() error {
	if d.tx == nil {
		return driver.ErrNotInTransaction
	}
	d.txDone = true
	return d.tx.Rollback()
}

func (d *DB) Close() error {
	if d.tx != nil {
		if !d.txDone {
			return d.tx.Rollback()
		}
		return nil
	}
	return d.sqlDb.Close()
}

func (d *DB) QuoteString(s string) string {
	return d.quoteWith(s, d.driver.backend.StringQuote())
}

func (d *DB) QuoteIdentifier(s string) string {
	return d.quoteWith(s, d.driver.backend.IdentifierQuote())
}

func (d *DB) quoteWith(s string, q byte) string {
	qu := string(q)
	var escaped string
	if q == '\'' {
		escaped = strings.Replace(s, "'", "''", -1)
	} else {
		escaped = strings.Replace(s, qu, "\\"+qu, -1)
	}
	return qu + escaped + qu
}

func (d *DB) DB() *sql.DB {
	return d.sqlDb
}

func (d *DB) Driver() *Driver {
	return d.driver
}

func (d *DB) Backend() Backend {
	return d.driver.backend
}
