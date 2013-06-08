package orm

import (
	"gondola/orm/driver"
)

type SortDirection int

const (
	// ASC is used for two purposes. First, it can be used for indicating the
	// sort direction in queries. It can also be used when creating
	// indexes to specify the ordering of the field. See the documentation
	// on DESC for further information.
	ASC SortDirection = driver.ASC
	// DESC is used for two purposes. First, it can be used for indicating the
	// sort direction in queries. It can also be used when creating
	// indexes to specify the ordering of the field. e.g.
	//     Index("A", "B", "C").Set(DESC, []string{"B"})
	// will create an index where A and C are sorted in the default order
	// (usually ascending) while B is sorted in descending order.
	DESC = driver.DESC
)
