package driver

// Capability indicates the capabilities of an
// ORM driver.
type Capability int

const (
	// No capabilities
	CAP_NONE Capability = 0
	// Can perform JOINs
	CAP_JOIN = 1 << iota
	// Can create transactions
	CAP_TRANSACTION
	// Can begin/commit/rollback a transaction
	CAP_BEGIN
	// Can automatically assign ids to rows
	CAP_AUTO_ID
	// Automatically assigned ids increase sequentially
	CAP_AUTO_INCREMENT
	// Provides eventual consistency rather than strong consistency
	CAP_EVENTUAL
	// Supports having a primary key
	CAP_PK
	// Primary key can be formed from multiple fields
	CAP_COMPOSITE_PK
	// Can have non-PK unique fields, enforce by the backend.
	CAP_UNIQUE
	// Can have database level defaults.
	CAP_DEFAULTS
	// Can have database level defaults for TEXT fields (unbounded strings).
	CAP_DEFAULTS_TEXT
)
