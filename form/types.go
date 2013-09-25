package form

type Type int

const (
	// <input type="text">
	TEXT Type = iota + 1
	// <input type="password">
	PASSWORD
	// <input type="hidden">
	HIDDEN
	// <input type="text">
	INTEGER
	// <input type="text">
	UINTEGER
	// <input type="text">
	FLOAT
	// <textarea>
	TEXTAREA
	// <input type="checkbox">
	BOOL
	// <input type="radio">
	CHOICES
	// <select>
	SELECT
)
