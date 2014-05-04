// +build !profile

package profile

const On = false

var ev = &Ev{}

type Ev struct{}

func (e *Ev) Note(format string, args ...interface{}) {}
func (e *Ev) End()                                    {}
func (e *Ev) AutoEnd()                                {}

func Begin()                                                             {}
func End()                                                               {}
func Profiling() bool                                                    { return false }
func Start(name string) *Ev                                              { return ev }
func Startf(name string, format string, args ...interface{}) *Ev         { return ev }
func HasEvent() bool                                                     { return false }
func Note(format string, args ...interface{})                            {}
func Profile(f func(), name string)                                      { f() }
func Profilef(f func(), name string, format string, args ...interface{}) { f() }
func Timings() []*Timing                                                 { return nil }
