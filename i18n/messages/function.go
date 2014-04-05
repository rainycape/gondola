package messages

type Function struct {
	// Qualified function name
	Name string
	// Wheter the function is a template function
	Template bool
	// Wheter the function has a context argument
	Context bool
	// Wheter the function has a plural form argument
	Plural bool
	// Position of the first translatable argument (0 indexed)
	Start int
}
