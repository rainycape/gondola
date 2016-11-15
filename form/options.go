package form

// Options specify the Form options at creation time.
type Options struct {
	// Render is the type which renders the form. If Renderer
	// is nil, DefaultRenderer() is used to obtain a new Renderer.
	// When writing a reusable package or app, is recommended to always
	// leave this field with a nil value, so the app user can import
	// a package which redefines the default Renderer to use the
	// frontend framework used in the final app
	// (like e.g. gnd.la/frontend/bootstrap3).
	Renderer Renderer
	// Fields lists the struct fields to include in the form. If empty,
	// all exported fields are included.
	Fields []string
}
