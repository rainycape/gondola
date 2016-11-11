package template

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// FuncTrait indicates the traits of a template function, which allow the
// template engine to pass additional arguments to them.
type FuncTrait int

const (
	// FuncTraitPure declares the function as pure (depends only on its arguments) and allows
	// evaluating at compile time if its arguments are constant.
	FuncTraitPure FuncTrait = 1 << iota
	// FuncTraitContext declares as a Context function. Context functions receive an additional
	// Context argument as passed to Template.ExecuteContext.
	// The Context argument type must be assignable from the type of the
	// Context used during execution. The Context argument is passed before the arguments
	// in the template function call. e.g.
	//
	//  func MyFunction(ctx *MyTemplateContext, someArg someType, someotherArg...) someThing
	//
	FuncTraitContext
	// FuncTraitState declares a State function. Note that a function might be both a Context
	// and a State function, with the State argument coming before the Context argument. The
	// State argument is a pointer to the execution state of the template, as represented by
	// the State type. State functions might use any method exported by State. e.g.
	//
	//  func MyStateFunction(s *State, someArg someType, someotherArg...) someThing
	//
	FuncTraitState
)

// HasTrait returns true iff ft has all traits in t
func (ft FuncTrait) HasTrait(t FuncTrait) bool {
	return ft&t == t
}

// nTraitArgs returns the number of extra arguments due to
// the traits.
func (ft FuncTrait) nTraitArgs() int {
	c := 0
	if ft.HasTrait(FuncTraitContext) {
		c++
	}
	if ft.HasTrait(FuncTraitState) {
		c++
	}
	return c
}

var (
	funcRegistry = make(map[string]*Func)
)

// Func represents a function which is available
// to be called from a template.
type Func struct {
	Name     string
	Fn       interface{}
	Traits   FuncTrait
	fval     reflect.Value
	variadic bool
	numIn    int
	fp       fastPath
}

func (f *Func) initialize() error {
	if f.fval.IsValid() {
		// Already initialized
		return nil
	}
	if f.Name == "" {
		return errors.New("empty template function name")
	}
	if f.Fn == nil {
		return fmt.Errorf("template function %q has no implementation", f.Name)
	}
	v := reflect.ValueOf(f.Fn)
	if v.Kind() != reflect.Func {
		return fmt.Errorf("template function %q is not a function, it's %s", f.Name, v.Type())
	}
	f.fval = v
	typ := v.Type()
	f.variadic = typ.IsVariadic()
	f.numIn = typ.NumIn()
	f.fp = newFastPath(v)
	return nil
}

func (f *Func) mustInitialize() *Func {
	if err := f.initialize(); err != nil {
		panic(err)
	}
	return f
}

type FuncOption func(Func) Func

// FuncPure applies the FuncTraitPure trait to the function
func FuncPure(f Func) Func {
	f.Traits |= FuncTraitPure
	return f
}

// RegisterFunc registers a template func and makes it available
// to all templates.
func RegisterFunc(name string, fn interface{}, opts ...FuncOption) {
	f := Func{
		Name: name,
		Fn:   fn,
	}
	for _, o := range opts {
		f = o(f)
	}
	funcRegistry[name] = &f
}

type FuncMap map[string]*Func

func (m FuncMap) asTemplateFuncMap() map[string]interface{} {
	tfm := make(map[string]interface{}, len(m))
	for k, v := range m {
		tfm[k] = v.Fn
	}
	return tfm
}

func makeFuncMap(fns []*Func) FuncMap {
	m := make(FuncMap, len(fns))
	for _, f := range fns {
		if m[f.Name] != nil {
			var names []string
			for _, f := range fns {
				names = append(names, f.Name)
			}
			panic(fmt.Errorf("duplicate function name %q (names are %v)", f.Name, names))
		}
		f.mustInitialize()
		m[f.Name] = f
	}
	return m
}

func makeFunc(fn interface{}, traits FuncTrait) *Func {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
		name = name[dot+1:]
	}
	return makeNamedFunc(fn, name, traits)
}

func makePureFunc(fn interface{}) *Func {
	return makeFunc(fn, FuncTraitPure)
}

func makeNamedFunc(fn interface{}, name string, traits FuncTrait) *Func {
	f := &Func{
		Name: name,
		Fn:   fn,
	}
	f.mustInitialize()
	return f
}

func funcDebugName(name string, fn reflect.Value) string {
	if fn.IsValid() && fn.Kind() == reflect.Func {
		if rf := runtime.FuncForPC(fn.Pointer()); rf != nil {
			file, line := rf.FileLine(fn.Pointer())
			return fmt.Sprintf("%s (%s @ %s:%d)", name, rf.Name(), file, line)
		}

	}
	return name
}

func convertTemplateFuncMap(fns map[string]interface{}) FuncMap {
	m := make(FuncMap, len(fns))
	for k, v := range fns {
		var f *Func
		if fn, ok := v.(*Func); ok {
			if fn.Name == "" {
				fn.Name = k
			}
			f = fn
		} else {
			f = &Func{Name: k, Fn: v}
		}
		f.mustInitialize()
		m[k] = f
	}
	return m
}
