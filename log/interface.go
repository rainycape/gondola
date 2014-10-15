package log

// Interface is the interface implemented by any logger in Gondola.
type Interface interface {
	// Debug formats its arguments like fmt.Print and records a
	// log message at the debug level.
	Debug(args ...interface{})
	// Debugf formats its arguments like fmt.Printf and records a
	// log message at the debug level.
	Debugf(format string, args ...interface{})

	// Info formats its arguments like fmt.Print and records a
	// log message at the info level.
	Info(args ...interface{})
	// Infof formats its arguments like fmt.Printf and records a
	// log message at the info level.
	Infof(format string, args ...interface{})

	// Warning formats its arguments like fmt.Print and records a
	// log message at the warning level.
	Warning(args ...interface{})
	// Warningf formats its arguments like fmt.Printf and records a
	// log message at the warning level.
	Warningf(format string, args ...interface{})

	// Error formats its arguments like fmt.Print and records a
	// log message at the error level.
	Error(args ...interface{})
	// Errorf formats its arguments like fmt.Printf and records a
	// log message at the error level.
	Errorf(format string, args ...interface{})
}
