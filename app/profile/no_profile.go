// +build !profile

package profile

const On = false

var ev = &Timed{}

type Timed struct{}

func (t *Timed) Note(title string, text string) *Timed                         { return t }
func (t *Timed) Notef(title string, format string, args ...interface{}) *Timed { return t }
func (t *Timed) End()                                                          {}
func (t *Timed) AutoEnd()                                                      {}

func ID() int                                                                          { return -1 }
func Begin()                                                                           {}
func End(_ int)                                                                        {}
func Profiling() bool                                                                  { return false }
func Start(name string) *Timed                                                         { return ev }
func Startf(name string, title string, format string, args ...interface{}) *Timed      { return ev }
func HasEvent() bool                                                                   { return false }
func Notef(title string, format string, args ...interface{})                           {}
func Profile(f func(), name string)                                                    { f() }
func Profilef(f func(), title string, name string, format string, args ...interface{}) { f() }
func Timings() []*Timing                                                               { return nil }
