package template

import (
	"errors"
	"fmt"
	"gnd.la/util/types"
	"io"
	"reflect"
	"text/template/parse"
)

var (
	errNoComplex = errors.New("complex number are not supported by the template compiler")
	zero         reflect.Value
	errType      = reflect.TypeOf((*error)(nil)).Elem()
	stringType   = reflect.TypeOf("")
	stringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	emptyType    = reflect.TypeOf((*interface{})(nil)).Elem()
)

// TODO: Remove variables inside if or with when exiting the scope

const (
	opNOP uint8 = iota
	opBOOL
	opFIELD
	opFLOAT
	opFUNC
	opINT
	opJMP
	opJMPT
	opJMPF
	opUINT
	opWB
	opDOT
	opPRINT
	opSTRING
	opPUSHDOT
	opPOPDOT
	opPOP
	opSETVAR
	opSTACKDOT
	opVAR
)

type valType uint32

type inst struct {
	op  uint8
	val valType
}

type variable struct {
	name  string
	value reflect.Value
}

type state struct {
	p     *Program
	w     io.Writer
	vars  []variable
	stack []reflect.Value
	dot   []reflect.Value
}

func (s *state) errorf(format string, args ...interface{}) {
}

func (s *state) pushVar(name string, value reflect.Value) {
	s.vars = append(s.vars, variable{name, value})
}

func (s *state) varMark() int {
	return len(s.vars)
}

func (s *state) popVar(mark int) {
	s.vars = s.vars[0:mark]
}

func (s *state) setVar(n int, value reflect.Value) {
	s.vars[len(s.vars)-n].value = value
}

func (s *state) varValue(name string) reflect.Value {
	for i := s.varMark() - 1; i >= 0; i-- {
		if s.vars[i].name == name {
			return s.vars[i].value
		}
	}
	s.errorf("undefined variable: %s", name)
	return zero
}

// call fn, remove its args from the stack and push the result
func (s *state) call(fn reflect.Value, name string, args int) error {
	//	fmt.Println("WILL CALL", name, args, len(s.stack), s.stack)
	pos := len(s.stack) - args
	in := s.stack[pos:]
	// arguments are in reverse order
	for ii := 0; ii < len(in)/2; ii++ {
		in[ii], in[len(in)-1-ii] = in[len(in)-1-ii], in[ii]
	}
	res := fn.Call(in)
	//	fmt.Println("CALLED", name, res)
	if len(res) == 2 && !res[1].IsNil() {
		return fmt.Errorf("error calling %q: %s", name, res[1].Interface())
	}
	s.stack = append(s.stack[:pos], stackable(res[0]))
	return nil
}

func (s *state) execute(name string, dot reflect.Value) error {
	code := s.p.code[name]
	s.dot = []reflect.Value{dot}
	if code == nil {
		return fmt.Errorf("template %q does not exist", name)
	}
	for ii := 0; ii < len(code); ii++ {
		v := code[ii]
		switch v.op {
		case opFIELD:
			var res reflect.Value
			p := len(s.stack) - 1
			top := s.stack[p]
			i := int(v.val & 0xFFFF)
			//			fmt.Println("FIELD", s.p.strings[int(v.val)])
			if top.IsValid() {
				if top.Kind() == reflect.Map && top.Type().Key().Kind() == reflect.String {
					k := s.p.rstrings[i]
					res = stackable(top.MapIndex(k))
					//					fmt.Println("KEYED", k, res, top.Interface())
				} else {
					name := s.p.strings[i]
					// get pointer methods and try to call a method by that name
					ptr := top
					if ptr.Kind() != reflect.Interface && ptr.CanAddr() {
						ptr = ptr.Addr()
					}
					if fn := ptr.MethodByName(name); fn.IsValid() {
						// when calling a function from a field, there will be
						// and extra argument at the top of the stack, either
						// the dot or the result of the last field lookup, so
						// we have to nuke it.
						s.stack = s.stack[:len(s.stack)-1]
						args := int(v.val >> 16)
						if err := s.call(fn, name, args); err != nil {
							return err
						}
						// s.call already puts the result in the stack
						break
					}
					// try to get a field by that name
					for top.Kind() == reflect.Ptr {
						if top.IsNil() {
							return fmt.Errorf("nil pointer evaluationg field %q on type %T", name, top.Interface())
						}
						top = top.Elem()
					}
					if top.Kind() != reflect.Struct {
						return fmt.Errorf("can't evaluate field on type %T", top.Interface())
					}
					res = top.FieldByName(name)
					if !res.IsValid() {
						// TODO: Check if the type has a pointer method which we couldn't
						// address, to provide a better error message
						return fmt.Errorf("%q is not a field of struct type %T", name, top.Interface())
					}
				}
			}
			// opFIELD overwrites the stack
			s.stack[p] = res
		case opFUNC:
			args := int(v.val >> 16)
			i := int(v.val & 0xFFFF)
			fn := s.p.funcs[i]
			// function existence is checked at compile time
			if err := s.call(fn.val, fn.name, args); err != nil {
				return err
			}
		case opVAR:
			name := s.p.strings[int(v.val)]
			s.stack = append(s.stack, s.varValue(name))
		case opJMP:
			ii += int(v.val)
		case opJMPF:
			p := len(s.stack)
			if p == 0 || !isTrue(s.stack[p-1]) {
				//				fmt.Println("FALSE JMP", v.val)
				ii += int(v.val)
			}
		case opSETVAR:
			name := s.p.strings[int(v.val)]
			s.pushVar(name, s.stack[len(s.stack)-1])
		case opBOOL, opFLOAT, opINT, opUINT:
			s.stack = append(s.stack, s.p.values[v.val])
		case opJMPT:
			p := len(s.stack)
			if p > 0 && isTrue(s.stack[p-1]) {
				ii += int(v.val)
			}
		case opPOP:
			if v.val == 0 {
				// POP all
				s.stack = s.stack[:0]
			} else {
				s.stack = s.stack[:len(s.stack)-int(v.val)]
			}
		case opPRINT:
			v := s.stack[len(s.stack)-1]
			val, ok := printableValue(v)
			if !ok {
				return fmt.Errorf("can't print value of type %s", v.Type())
			}
			if _, err := fmt.Fprint(s.w, val); err != nil {
				return err
			}
		case opPUSHDOT:
			s.dot = append(s.dot, dot)
			dot = s.stack[len(s.stack)-1]
		case opPOPDOT:
			p := len(s.dot) - 1
			dot = s.dot[p]
			s.dot = s.dot[:p]
		case opSTACKDOT:
			s.stack = append(s.stack, stackable(dot))
		case opSTRING:
			s.stack = append(s.stack, s.p.rstrings[int(v.val)])
		case opWB:
			if _, err := s.w.Write(s.p.bs[int(v.val)]); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid opcode %d", v.op)
		}
	}
	return nil
}

type fn struct {
	name     string
	val      reflect.Value
	variadic bool
	numIn    int
}

type Program struct {
	tmpl     *Template
	funcs    []*fn
	strings  []string
	rstrings []reflect.Value
	values   []reflect.Value
	bs       [][]byte
	code     map[string][]inst
	// used only during compilation
	buf []inst
	cmd []int
}

func (p *Program) inst(op uint8, val valType) {
	p.buf = append(p.buf, inst{op: op, val: val})
}

func (p *Program) addString(s string) valType {
	for ii, v := range p.strings {
		if v == s {
			return valType(ii)
		}
	}
	p.strings = append(p.strings, s)
	p.rstrings = append(p.rstrings, reflect.ValueOf(s))
	return valType(len(p.strings) - 1)
}

func (p *Program) addFunc(f interface{}, name string) valType {
	for ii, v := range p.funcs {
		if v.name == name {
			return valType(ii)
		}
	}
	// TODO: Check it's really a reflect.Func
	val := reflect.ValueOf(f)
	p.funcs = append(p.funcs, &fn{
		name:     name,
		val:      val,
		variadic: val.Type().IsVariadic(),
		numIn:    val.Type().NumIn(),
	})
	return valType(len(p.funcs) - 1)
}

func (p *Program) addValue(v interface{}) valType {
	p.values = append(p.values, reflect.ValueOf(v))
	return valType(len(p.values) - 1)
}

func (p *Program) addWB(b []byte) {
	pos := len(p.bs)
	p.bs = append(p.bs, b)
	p.inst(opWB, valType(pos))
}

func (p *Program) addSTRING(s string) {
	p.inst(opSTRING, p.addString(s))
}

func (p *Program) addFIELD(name string) {
	var args int
	if len(p.cmd) > 0 {
		args = p.cmd[len(p.cmd)-1] - 1 // first arg is the FieldNode
	}
	val := valType(args<<16) | p.addString(name)
	p.inst(opFIELD, val)
}

func (p *Program) walkBranch(nt parse.NodeType, b *parse.BranchNode) error {
	if err := p.walk(b.Pipe); err != nil {
		return err
	}
	// Save buf
	buf := p.buf
	p.buf = nil
	if err := p.walk(b.List); err != nil {
		return err
	}
	list := append([]inst{{op: opPOP}}, p.buf...)
	p.buf = nil
	if err := p.walk(b.ElseList); err != nil {
		return err
	}
	elseList := append([]inst{{op: opPOP}}, p.buf...)
	skip := len(list)
	if len(elseList) > 0 {
		skip += 1
	}
	if nt == parse.NodeWith {
		skip += 2
	}
	p.buf = buf
	p.inst(opJMPF, valType(skip))
	if nt == parse.NodeWith {
		p.inst(opPUSHDOT, 0)
	}
	p.buf = append(p.buf, list...)
	if nt == parse.NodeWith {
		p.inst(opPOPDOT, 0)
	}
	if c := len(elseList); c > 0 {
		p.inst(opJMP, valType(c))
		p.buf = append(p.buf, elseList...)
	}
	return nil
}

func (p *Program) walk(n parse.Node) error {
	//	fmt.Printf("NODE %T %+v\n", n, n)
	switch x := n.(type) {
	case *parse.ActionNode:
		if err := p.walk(x.Pipe); err != nil {
			return err
		}
		if len(x.Pipe.Decl) == 0 {
			p.inst(opPRINT, 0)
		}
		p.inst(opPOP, 0)
	case *parse.BoolNode:
		p.inst(opBOOL, p.addValue(x.True))
	case *parse.CommandNode:
		p.cmd = append(p.cmd, len(x.Args))
		// Command nodes are pushed on reverse order, so they are
		// evaluated from right to left. If we encounter a function
		// while executing it, we can just grab the arguments from the stack
		for ii := len(x.Args) - 1; ii >= 0; ii-- {
			node := x.Args[ii]
			if err := p.walk(node); err != nil {
				return err
			}
		}
		/*} else {
			for _, node := range x.Args {
				if err := p.walk(node); err != nil {
					return err
				}
			}
		}*/
		p.cmd = p.cmd[:len(p.cmd)-1]
	case *parse.DotNode:
		p.inst(opDOT, 0)
	case *parse.FieldNode:
		p.inst(opSTACKDOT, 0)
		for _, v := range x.Ident {
			p.addFIELD(v)
		}
	case *parse.IdentifierNode:
		if len(p.cmd) == 0 {
			return fmt.Errorf("identifier %q outside of command?", x.Ident)
		}
		args := p.cmd[len(p.cmd)-1] - 1 // first arg is identifier
		fn := p.tmpl.funcMap[x.Ident]
		if fn == nil {
			return fmt.Errorf("undefined function %q", x.Ident)
		}
		val := valType(args<<16) | p.addFunc(fn, x.Ident)
		p.inst(opFUNC, val)
	case *parse.IfNode:
		if err := p.walkBranch(parse.NodeIf, &x.BranchNode); err != nil {
			return err
		}
	case *parse.ListNode:
		for _, node := range x.Nodes {
			if err := p.walk(node); err != nil {
				return err
			}
		}
	case *parse.NumberNode:
		switch {
		case x.IsComplex:
			return errNoComplex
		case x.IsFloat:
			p.inst(opFLOAT, p.addValue(x.Float64))
		case x.IsInt:
			p.inst(opINT, p.addValue(x.Int64))
		case x.IsUint:
			p.inst(opUINT, p.addValue(x.Uint64))
		}
	case *parse.PipeNode:
		for _, v := range x.Cmds {
			if err := p.walk(v); err != nil {
				return err
			}
		}
		for _, variable := range x.Decl {
			// Remove $
			p.inst(opSETVAR, p.addString(variable.Ident[0][1:]))
		}
	case *parse.StringNode:
		p.addSTRING(x.Text)
	case *parse.TextNode:
		p.addWB(x.Text)
	case *parse.VariableNode:
		// Remove $ sign
		p.inst(opVAR, p.addString(x.Ident[0][1:]))
		for _, v := range x.Ident[1:] {
			p.addFIELD(v)
		}
	case *parse.WithNode:
		if err := p.walkBranch(parse.NodeWith, &x.BranchNode); err != nil {
			return err
		}
	default:
		return fmt.Errorf("can't compile node %T", n)
	}
	return nil
}

func (p *Program) Execute(w io.Writer, data interface{}) error {
	s := &state{
		p: p,
		w: w,
	}
	return s.execute(p.tmpl.Root(), reflect.ValueOf(data))
}

func NewProgram(tmpl *Template) (*Program, error) {
	p := &Program{tmpl: tmpl, code: make(map[string][]inst)}
	for k, v := range tmpl.Trees {
		if err := p.walk(v.Root); err != nil {
			return nil, err
		}
		p.code[k] = p.buf
		p.buf = nil
	}
	return p, nil
}

func isTrue(v reflect.Value) bool {
	t, _ := types.IsTrueVal(v)
	return t
}

func printableValue(v reflect.Value) (interface{}, bool) {
	if v.Kind() == reflect.Ptr {
		v, _ = indirect(v) // fmt.Fprint handles nil.
	}
	if !v.IsValid() {
		return "<no value>", true
	}

	if !isPrintable(v.Type()) {
		if v.CanAddr() && isPrintable(reflect.PtrTo(v.Type())) {
			v = v.Addr()
		} else {
			switch v.Kind() {
			case reflect.Chan, reflect.Func:
				return nil, false
			}
		}
	}
	return v.Interface(), true
}

func isPrintable(typ reflect.Type) bool {
	return typ.Implements(errType) || typ.Implements(stringerType)
}

func stackable(v reflect.Value) reflect.Value {
	if v.IsValid() && v.Type() == emptyType {
		v = reflect.ValueOf(v.Interface())
	}
	return v
}
