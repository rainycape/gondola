package admin

const (
	typBool = iota + 1
	typInt
	typString
)

// Flag is a opaque type used to represent a flag for
// a command. Use the BoolFlag(), IntFlag() and
// StringFlag() functions to create a Flag. You can also
// use the convenience function Flags() to create a
// slice with several flags.
type Flag struct {
	name string
	help string
	typ  int
	def  interface{}
}

// Options is used to specify the command options when
// registering it.
type Options struct {
	// The name of the command. If no name is provided,
	// it will be obtained from the function name, transforming
	// its name from camel case to words separated by a '-'
	Name string
	// The help string that will be printed for this command.
	Help string
	// Usage is printed just after the Help, prepending the command to it.
	Usage string
	// Any flags this command might accept. Use the convenience
	// functions to define them.
	Flags []*Flag
}

// Flags is a convenience function which returns the received flags as a slice.
func Flags(flags ...*Flag) []*Flag {
	return flags
}

func makeFlag(name string, help string, typ int, def interface{}) *Flag {
	return &Flag{
		name: name,
		help: help,
		typ:  typ,
		def:  def,
	}
}

// BoolFlag returns a flag of type bool with the given name,
// default value and help.
func BoolFlag(name string, def bool, help string) *Flag {
	return makeFlag(name, help, typBool, def)
}

// IntFlag returns a flag of type int with the given name,
// default value and help.
func IntFlag(name string, def int, help string) *Flag {
	return makeFlag(name, help, typInt, def)
}

// StringFlag returns a flag of type string with the given name,
// default value and help.
func StringFlag(name string, def string, help string) *Flag {
	return makeFlag(name, help, typString, def)
}
