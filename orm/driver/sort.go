package driver

type SortDirection int

const (
	// These constants are documented in the gnd.la/orm package
	DESC SortDirection = -1
	ASC                = 1
)

type Sort interface {
	Field() string
	Direction() SortDirection
}
