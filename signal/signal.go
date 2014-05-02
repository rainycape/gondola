package signal

import (
	"errors"
	"fmt"
	"reflect"

	"gnd.la/internal/runtimeutil"
	"gnd.la/log"
)

var (
	signals = map[string][]*reflect.Value{}
)

type Token struct {
	val *reflect.Value
}

// Listen adds a new listener for the given signal name. The
// second argument must be a function which accepts either:
//
// - no paremeters
// - 1 parameter, which must be of type string
// - 2 parameters, the first one must be string and the second one, interface{}
//
// If the function does not match the required constraints, Listen
// will panic.
//
// The function will be called whenever the signal is emitted. The
// returned value is the token, which is required to unregister this
// listener. If you don't need to unregister it, you can safely ignore
// the first returned value.
func Listen(name string, f interface{}) *Token {
	tok, err := listen(name, f)
	if err != nil {
		panic(err)
	}
	return tok
}

func listen(name string, f interface{}) (*Token, error) {
	if name == "" {
		return nil, errors.New("signal name can't be empty")
	}
	val := reflect.ValueOf(f)
	if err := checkListener(val); err != nil {
		return nil, err
	}
	signals[name] = append(signals[name], &val)
	return &Token{&val}, nil
}

// Stop removes a listener, previously registered using Listen. The
// first argument indicates the signal name. If it's empty, the listener will
// be removed for all the signal. The second argument is the token returned by
// Listen(). If it's empty, all the listeners for the given signals will be
// removed.
func Stop(name string, t *Token) {
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
			v.Call(params[:v.Type().NumIn()])
		}
	}
}

func removeToken(s map[string][]*reflect.Value, name string, t *Token) {
	rec := s[name]
	if rec != nil {
		if t == nil {
			delete(s, name)
			return
		}
		r := t.val
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

func checkListener(val reflect.Value) error {
	if !val.IsValid() {
		return errors.New("listener is not a valid value - probably nil")
	}
	if val.Kind() != reflect.Func {
		return fmt.Errorf("listener is of type %s, not function", val.Type())
	}
	vt := val.Type()
	if vt.NumOut() > 0 {
		return fmt.Errorf("listeners can't return arguments, %s returns %d", runtimeutil.FuncName(val.Interface()), vt.NumOut())
	}
	switch vt.NumIn() {
	case 2:
		emptyType := reflect.TypeOf((*interface{})(nil)).Elem()
		typ := vt.In(1)
		if typ != emptyType {
			return fmt.Errorf("%s second argument must of type %s, not %s", runtimeutil.FuncName(val.Interface()), emptyType, typ)
		}
		fallthrough
	case 1:
		stringType := reflect.TypeOf("")
		typ := vt.In(0)
		if typ != stringType {
			return fmt.Errorf("%s first argument must of type %s, not %s", runtimeutil.FuncName(val.Interface()), stringType, typ)
		}
	case 0:
	default:
		return fmt.Errorf("listeners can accept 0 or 1 or 2 arguments, %s accepts %d", runtimeutil.FuncName(val.Interface()), vt.NumIn())
	}
	return nil
}
