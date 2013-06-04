package orm

type Options struct {
	Name      string
	Indexes   []*Index
	Relations []*Relation
}
