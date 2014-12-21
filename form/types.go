package form

type Type int

const (
	// <input type="text">
	TEXT Type = iota + 1
	// <input type="password">
	PASSWORD
	// <input type="email">
	EMAIL
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
	// <input type="file">
	FILE
)

// HasChoices returns wheter the type has multiple
// choices, which corresponds to RADIO and SELECT
// elements.
func (t Type) HasChoices() bool {
	return t == RADIO || t == SELECT
}
