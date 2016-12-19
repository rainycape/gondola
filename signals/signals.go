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
// used to stop listening for the signal by calling its Remove method.
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
