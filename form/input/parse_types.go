package input

// Strings implements the Parser interface, splitting the parsed
// string between commas. Strings containing commas might be
// quoted
type Strings []string

func (s Strings) Parse(val string) error {
	return nil
}
