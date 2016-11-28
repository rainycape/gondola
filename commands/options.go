package commands

const (
	typBool = iota + 1
	typInt
	typString
)

// cmdFlag is a opaque type used to represent a flag for
// a command. Use the BoolFlag(), IntFlag() and
// StringFlag() functions to create a Flag. You can also
// use the convenience function Flags() to create a
// slice with several flags.
type cmdFlag struct {
	name string
	help string
	typ  int
	def  interface{}
}

// Option is a function type which sets one or several command options.
// Use the Option implementations in this package.
type Option func(opts options) options

// options is used to specify the command options when
// registering it.
type options struct {
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
	Flags []*cmdFlag
}

func makeFlag(name string, help string, typ int, def interface{}) Option {
	fl := &cmdFlag{
		name: name,
		help: help,
		typ:  typ,
		def:  def,
	}
	return func(opts options) options {
		opts.Flags = append(opts.Flags, fl)
		return opts
	}
}

// BoolFlag adds a flag of type bool with the given name,
// default value and help.
func BoolFlag(name string, def bool, help string) Option {
	return makeFlag(name, help, typBool, def)
}

// IntFlag adds a flag of type int with the given name,
// default value and help.
func IntFlag(name string, def int, help string) Option {
	return makeFlag(name, help, typInt, def)
}

// StringFlag adds a flag of type string with the given name,
// default value and help.
func StringFlag(name string, def string, help string) Option {
	return makeFlag(name, help, typString, def)
}

// Name sets the name of the command. If no name is provided,
// it will be obtained from the function name, transforming
// its name from camel case to words separated by a '-'
func Name(name string) Option {
	return func(opts options) options {
		opts.Name = name
		return opts
	}
}

// Help sets help string that will be printed for a command.
func Help(help string) Option {
	return func(opts options) options {
		opts.Help = help
		return opts
	}
}

// Usage sets the command usage help string. Usage is printed just after
// the help, prepending the command to it.
func Usage(usage string) Option {
	return func(opts options) options {
		opts.Usage = usage
		return opts
	}
}
