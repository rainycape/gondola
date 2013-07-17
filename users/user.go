package users

// User is the interface implemented by any struct
// that can be used to represent a user in a Gondola
// app.
type User interface {
	// Returns the numeric id of the user
	Id() int64
}
