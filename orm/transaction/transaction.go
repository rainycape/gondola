// Package transaction contains constants for controlling the behavior
// of database transactions.
// Most constants are only interpreted by one driver and ignored by the
// rest. To start a transaction with a given set of options call
// BeginOptions() on a gondola/orm.Orm instance.
package transaction

type Options int64

const (
	NONE Options = 0
	// Options used by sqlite. See http://www.sqlite.org/lang_transaction.html
	// These two options are mutually exclusive, the first one takes precendence.
	IMMEDIATE = 1 << 0
	EXCLUSIVE = 1 << 1
	// Options used by postgres. See http://www.postgresql.org/docs/9.2/static/sql-set-transaction.html
	READ_ONLY       = 1 << 2
	REPEATABLE_READ = 1 << 3
	SERIALIZABLE    = 1 << 4
	DEFERRABLE      = 1 << 5
)
