package signals

import (
	"sync"

	"gnd.la/log"
)

// Handler is a function type which receives a signal emitted using this package.
// The first argument is the signal name, while the second one is a signal dependent
// arbitrary parameter. Signal emitters should clearly document what listeners can
// expect to receive in this parameter.
type Handler func(name string, data interface{})

var (
	mu        sync.RWMutex
	listeners = make(map[string][]*listener)
)

// Listener is the interface returned by Listen. Call Remove to stop listening
// for the signal.
type Listener interface {
	Remove()
}

type listener struct {
	Name    string
	Handler Handler
}

func (listener *listener) Remove() {
	mu.Lock()
	ref := listeners[listener.Name]
	for ii, v := range ref {
		if listener == v {
			copy(ref[ii:], ref[ii+1:])
			ref[len(ref)-1] = nil
			ref = ref[:len(ref)-1]
			break
		}
	}
	if len(ref) == 0 {
		delete(listeners, listener.Name)
	} else {
		listeners[listener.Name] = ref
	}
	mu.Unlock()
}

// Listen adds a new listener for the given signal name. The returned Listener can
// be used to stop listening for the signal by calling its Remove method.
func Listen(name string, handler Handler) Listener {
	listener := &listener{
		Name:    name,
		Handler: handler,
	}
	mu.Lock()
	listeners[name] = append(listeners[name], listener)
	mu.Unlock()
	return listener
}

// Emit calls all the listeners for the given signal.
func Emit(name string, data interface{}) {
	log.Debugf("Emitting signal %s with data %T", name, data)
	mu.RLock()
	ref := listeners[name]
	cpy := make([]*listener, len(ref))
	copy(cpy, ref)
	mu.RUnlock()
	for _, listener := range cpy {
		listener.Handler(name, data)
	}
}

// Signal is a conveniency type which allows callers to listen and (if desired)
// emit a given signal without providing access to the signal name itself.
// This allows better encapsulation as well as a simple way to implement
// type safesignals. See the package documentation for a complete example.
type Signal struct {
	name string
	listeners
}

// New returns a new Signal
func New(name string) *Signal {
	return &Signal{
		name: name,
	}
}

// Listen is a shorthand for Listen(signame, handler), where signame
// is the signal name received in New().
func (s *Signal) Listen(handler func(interface{})) Listener {
	return Listen(s.name, func(_ string, data interface{}) {
		handler(data)
	})
}

// Emit is a shorthand for Emit(signame, data), where signame
// is the signal name received in New().
func (s *Signal) Emit(data interface{}) {
	Emit(s.name, data)
}
