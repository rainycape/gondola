// +build !debug

package debug

const On = false

var ev = &Ev{}

type Ev struct{}

func (e *Ev) Note(format string, args ...interface{}) {}
func (e *Ev) End()                                    {}

func Start(name string) *Ev                                      { return ev }
func Startf(name string, format string, args ...interface{}) *Ev { return ev }
func HasEvent() bool                                             { return false }
func Note(format string, args ...interface{})                    {}
func End()                                                       {}
func Timings() []*Timing                                         { return nil }
