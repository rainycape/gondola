package loaders

// DirLoader is the interface implemented by a
// loader which loads the resources from a given
// directory.
type DirLoader interface {
	Loader
	// The root directory for the loaded resources.
	Dir() string
}
