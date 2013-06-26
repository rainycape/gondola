package signal

import (
	"errors"
	"gondola/log"
	"reflect"
)

var (
	signals      = map[string][]*reflect.Value{}
	ErrEmptyName = errors.New("signal name can't be empty")
)

type Token *reflect.Value

// Register adds a new listener for the given signal name. The
// second argument must be a function which accepts either:
// - no paremeters
// - 1 parameter, which must be of type string
// - 2 parameters, the first one must be string and the second one, interface{}
// The function will be called whenever the signal is emitted. The first
// returned value is the token, which is required to unregister this
// listener. If you don't need to unregister it, you can safely ignore
// the first returned value.
func Register(name string, f interface{}) (Token, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	if _, ok := signals[name]; !ok {
		signals[name] = make([]*reflect.Value, 0)
	}
	val := reflect.ValueOf(f)
	signals[name] = append(signals[name], &val)
	return Token(&val), nil
}

// MustRegister works like Register, but panics if there's an error.
func MustRegister(name string, f interface{}) Token {
	t, err := Register(name, f)
	if err != nil {
		panic(err)
	}
	return t
}

// Unregister removes a listener, previously registered using Register. The
// first argument indicates the signal name. If it's empty, the listener will
// be removed for all the signal. The second argument is the token returned by
// Register(). If it's empty, all the listeners for the given signals will be
// removed.
func Unregister(name string, t Token) {
	if name == "" {
		for k := range signals {
			removeToken(signals, k, t)
		}
	} else {
		removeToken(signals, name, t)
	}
}

// Emit calls all the listeners for the given signal.
func Emit(name string, object interface{}) {
	log.Debugf("Emitting signal %s with object %+v", name, object)
	if rec := signals[name]; rec != nil {
		params := []reflect.Value{reflect.ValueOf(name), reflect.ValueOf(object)}
		for _, v := range rec {
			v.Call(params)
		}
	}
}

func removeToken(s map[string][]*reflect.Value, name string, t Token) {
	rec := s[name]
	if rec != nil {
		if t == nil {
			delete(s, name)
			return
		}
		r := (*reflect.Value)(t)
		for {
			idx := -1
			for ii, v := range rec {
				if v == r {
					idx = ii
					break
				}
			}
			if idx >= 0 {
				rec[len(rec)-1], rec[idx], rec = nil, rec[len(rec)-1], rec[:len(rec)-1]
				continue
			}
			break
		}
		if len(rec) == 0 {
			delete(s, name)
		}
	}
}
