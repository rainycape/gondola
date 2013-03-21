package errors

// NotFound raises a NotFound error and stops
// execution of the current handler
func NotFound() {
	KindNotFound("")
}

// KindNotFound raises a NotFound indicating
// what was not found (.e.g "Article not found)
// and stops execution of the current handler
func KindNotFound(kind string) {
	panic(&NotFoundError{kind})
}

// MissingParameter raises a MissingParameter error
// with the given name and stops execution of the
// current handler
func MissingParameter(name string) {
	panic(&MissingParameterError{name})
}
