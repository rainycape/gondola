package index

const (
	// ASC is used when creating indexes, to specify the ordering
	// of the field. See the documentation on DESC for further information.
	ASC = iota + 1
	// DESC is used when creating indexes to specify the ordering of the field. e.g.
	//	New("A", "B", "C").Set(DESC, []string{"B"})
	// will create an index where A and C are sorted in the default order
	// (usually ascending) while B is sorted in descending order.
	DESC
)
