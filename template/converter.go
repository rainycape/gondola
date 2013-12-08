package template

var (
	converters = map[string]Converter{}
)

// Converter represents a function which converts a template
// source with a given extension into the source of an  HTML
// template. Use RegisterConverter to  register your own converters.
type Converter func([]byte) ([]byte, error)

// RegisterConverter registers a template converter for the
// given extension. If there's already a converter for the
// given extension, it's overwritten by the new one.
func RegisterConverter(ext string, c Converter) {
	if len(ext) > 0 && ext[0] != '.' {
		ext = "." + ext
	}
	converters[ext] = c
}
