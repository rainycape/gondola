package signals

import (
	"runtime"
	"strings"
	"sync"

	"gnd.la/log"
)

// Listener is the interface returned by Listen. Call Remove to stop listening
// for the signal.
type Listener interface {
	Remove()
}

type listener struct {
	Signal  *Signal
	Handler func(data interface{})
}

func (listener *listener) Remove() {
	listener.Signal.removeListener(listener)
}

// Signal allows emitting and receiving signals. Use New to create
// a new Signal. Note that Signal should usually be wrapped in
// another type which emits/listens for a typed signal. See
// the package documentation for details.
type Signal struct {
	mu        sync.RWMutex
	name      string
	listeners []*listener
}

// New returns a new Signal. The name parameter will be combined
// with the caller package name. e.g. ("mysignal" from package
// example.com/pkg/subpkg will become "example.com/pkg/subpkg.mysignal").
// Note that the name is optional and right now is only used in debug
// messages.
func New(name ...string) *Signal {
	var fullName string
	if len(name) > 0 {
		// Prepend package name
		suffix := strings.Join(name, ".")
		if pkgName := getCallerPackageName(); pkgName != "" {
			fullName = pkgName + "." + suffix
		} else {
			fullName = suffix
		}
	}
	return &Signal{
		name: fullName,
	}
}

// Listen is a shorthand for Listen(signame, handler), where signame
// is the signal name received in New().
func (s *Signal) Listen(handler func(interface{})) Listener {
	s.mu.Lock()
	listener := &listener{
		Signal:  s,
		Handler: handler,
	}
	s.listeners = append(s.listeners, listener)
	s.mu.Unlock()
	return listener
}

// Emit emits this signal, calling all registered listeners. Note
// that this function should only be called from a typed Signal
// wrapping it. See the package documentation for examples.
func (s *Signal) Emit(data interface{}) {
	s.mu.RLock()
	if c := len(s.listeners); c > 0 {
		listeners := make([]*listener, len(s.listeners))
		copy(listeners, s.listeners)
		// Don't hold the lock while invoking the
		// listeners, otherwise calling Remove()
		// from a handler would cause a deadlock.
		s.mu.RUnlock()
		if s.name != "" {
			log.Debugf("emitting signal %v (%d listeners)", s.name, c)
		}
		for _, v := range listeners {
			v.Handler(data)
		}
	} else {
		s.mu.RUnlock()
	}
}

func (s *Signal) removeListener(listener *listener) {
	s.mu.Lock()
	for ii, v := range s.listeners {
		if listener == v {
			copy(s.listeners[ii:], s.listeners[ii+1:])
			s.listeners[len(s.listeners)-1] = nil
			s.listeners = s.listeners[:len(s.listeners)-1]
			break
		}
	}
	s.mu.Unlock()
}

func getCallerPackageName() string {
	pc, _, _, ok := runtime.Caller(2)
	if ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			fname := fn.Name()
			parts := strings.Split(fname, ".")
			// Func name is the last part, the rest
			// is the package.
			return strings.Join(parts[:len(parts)-1], ".")
		}
	}
	return ""
}
