package orm

type Relation struct {
	// Field name in the origin
	From string
	// Field name in the target
	To string
	// Target model name
	Target string
}
