package app

import (
	"fmt"
	"reflect"

	"gnd.la/internal/runtimeutil"
)

type namespace struct {
	vars       map[string]interface{}
	funcs      map[string]reflect.Value
	namespaces map[string]*namespace
}

func (ns *namespace) hasVar(key string) bool {
	if _, ok := ns.vars[key]; ok {
		return true
	}
	if _, ok := ns.funcs[key]; ok {
		return true
	}
	return false
}

func (ns *namespace) add(vars map[string]interface{}) error {
	inType := reflect.TypeOf((*Context)(nil))
	for k, v := range vars {
		if isReservedVariable(k) {
			return fmt.Errorf("variable %q is reserved", k)
		}
		// Check that no variables shadow a namespace
		if ns.namespaces[k] != nil {
			return fmt.Errorf("can't add variable %q, there's already a namespace with that name", k)
		}
		if t := reflect.TypeOf(v); t.Kind() == reflect.Func {
			if t.NumIn() != 0 && (t.NumIn() != 1 || t.In(0) != inType) {
				return fmt.Errorf("template variable functions must receive either no arguments or a single %s argument", inType)
			}
			if t.NumOut() > 2 {
				return fmt.Errorf("template variable functions must return at most 2 arguments")
			}
			if t.NumOut() == 2 {
				o := t.Out(1)
				if o.Kind() != reflect.Interface || o.Name() != "error" {
					return fmt.Errorf("template variable functions must return an error as their second argument")
				}
			}
			// Check that func doesn't shadow a var
			if _, ok := ns.vars[k]; ok {
				return fmt.Errorf("can't add function %q, there's already a variable with that name", k)
			}
			if ns.funcs == nil {
				ns.funcs = make(map[string]reflect.Value)
			}
			ns.funcs[k] = reflect.ValueOf(v)
		} else {
			// Check that var doesn't shadow a func
			if _, ok := ns.funcs[k]; ok {
				return fmt.Errorf("can't add variable %q, there's already a function with that name", k)
			}
			if ns.vars == nil {
				ns.vars = make(map[string]interface{})
			}
			ns.vars[k] = v
		}
	}
	return nil
}

func (ns *namespace) addNs(name string, ans *namespace) error {
	// Check that the namespace does not shadow any variable
	if ns.hasVar(name) {
		return fmt.Errorf("can't add namespace %q because there's already a variable with that name", name)
	}
	if ns.namespaces == nil {
		ns.namespaces = make(map[string]*namespace)
	}
	ns.namespaces[name] = ans
	return nil
}

func (ns *namespace) eval(ctx *Context) (m map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			var pe error
			if e, ok := r.(error); ok {
				pe = e
			} else {
				pe = fmt.Errorf("%v", r)
			}
			err = pe
			if file, line, ok := runtimeutil.PanicLocation(); ok {
				err = fmt.Errorf("%v (at %s:%d)", pe, file, line)
			}
		}
	}()
	m = make(map[string]interface{}, len(ns.vars)+len(ns.funcs)+len(ns.namespaces)+2)
	for k, v := range ns.vars {
		m[k] = v
	}
	in := []reflect.Value{reflect.ValueOf(ctx)}
	for k, v := range ns.funcs {
		var out []reflect.Value
		if v.Type().NumIn() == 0 {
			out = v.Call(nil)
		} else {
			if ctx == nil {
				m[k] = nil
				continue
			}
			out = v.Call(in)
		}
		if len(out) == 2 && !out[1].IsNil() {
			return nil, out[1].Interface().(error)
		}
		m[k] = out[0].Interface()
	}
	m["Ctx"] = ctx
	if ctx != nil {
		m["Request"] = ctx.R
	}
	for k, v := range ns.namespaces {
		if m[k], err = v.eval(ctx); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func isReservedVariable(va string) bool {
	for _, v := range reservedVariables {
		if v == va {
			return true
		}
	}
	return false
}
