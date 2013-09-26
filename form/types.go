package form

type Type int

const (
	// <input type="text">
	TEXT Type = iota + 1
	// <input type="password">
	PASSWORD
	// <input type="hidden">
	HIDDEN
	// <textarea>
	TEXTAREA
	// <input type="checkbox">
	CHECKBOX
	// <input type="radio">
	RADIO
	// <select>
	SELECT
)
