package table

// Translation is the runtime representation
// of a translation loaded from a table.
type Translation struct {
	Context  *string
	Singular *string
	Plural   *string
	Plurals  map[int]string
}
