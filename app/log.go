package app

// nullLogger logs everything to /dev/null
type nullLogger struct {
}

func (n nullLogger) Debug(args ...interface{})                 {}
func (n nullLogger) Debugf(format string, args ...interface{}) {}

func (n nullLogger) Info(args ...interface{})                 {}
func (n nullLogger) Infof(format string, args ...interface{}) {}

func (n nullLogger) Warning(args ...interface{})                 {}
func (n nullLogger) Warningf(format string, args ...interface{}) {}

func (n nullLogger) Error(args ...interface{})                 {}
func (n nullLogger) Errorf(format string, args ...interface{}) {}
