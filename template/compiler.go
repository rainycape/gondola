package template

import (
	"errors"
	"fmt"
	"gnd.la/util/types"
	"io"
	"math"
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
	opSTACKDOT
)

type inst struct {
	op  uint8
	val uint64
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
	//	fmt.Println("WILL CALL", name, args)
	pos := len(s.stack) - args
	in := s.stack[pos:]
	// arguments are in reverse order
	for ii := 0; ii < len(in)/2; ii++ {
		in[ii], in[len(in)-1-ii] = in[len(in)-1-ii], in[ii]
	}
	//	fmt.Println("WILL CALL", name, in)
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
		case opBOOL:
			val := false
			if v.val > 0 {
				val = true
			}
			s.stack = append(s.stack, reflect.ValueOf(val))
		case opFIELD:
			var res reflect.Value
			p := len(s.stack) - 1
			top := s.stack[p]
			//			fmt.Println("FIELD", s.p.strings[int(v.val)])
			if top.IsValid() {
				if top.Kind() == reflect.Map && top.Type().Key() == stringType {
					k := s.p.rstrings[int(v.val)]
					res = stackable(top.MapIndex(k))
					//					fmt.Println("KEYED", k, res, top.Interface())
				}
			}
			// opFIELD overwrites the stack
			s.stack[p] = res
		case opFUNC:
			args := int(v.val >> 32)
			i := int(v.val & 0xFFFFFFFF)
			name := s.p.strings[i]
			fn := s.p.tmpl.funcMap[name]
			if fn == nil {
				return fmt.Errorf("undefined function %q", name)
			}
			if err := s.call(reflect.ValueOf(fn), name, args); err != nil {
				return err
			}
		case opFLOAT:
			s.stack = append(s.stack, reflect.ValueOf(math.Float64frombits(v.val)))
		case opINT:
			s.stack = append(s.stack, reflect.ValueOf(int64(v.val)))
		case opJMP:
			ii += int(v.val)
		case opJMPF:
			p := len(s.stack)
			if p == 0 || !isTrue(s.stack[p-1]) {
				//				fmt.Println("FALSE JMP", v.val)
				ii += int(v.val)
			}
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
		case opUINT:
			s.stack = append(s.stack, reflect.ValueOf(v.val))
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

type Program struct {
	tmpl     *Template
	strings  []string
	rstrings []reflect.Value
	bs       [][]byte
	code     map[string][]inst
	// used only during compilation
	buf []inst
	cmd []int
}

func (p *Program) inst(op uint8, val uint64) {
	p.buf = append(p.buf, inst{op: op, val: val})
}

func (p *Program) addString(s string) uint64 {
	pos := len(p.strings)
	p.strings = append(p.strings, s)
	p.rstrings = append(p.rstrings, reflect.ValueOf(s))
	return uint64(pos)
}

func (p *Program) addWB(b []byte) {
	pos := len(p.bs)
	p.bs = append(p.bs, b)
	p.inst(opWB, uint64(pos))
}

func (p *Program) addSTRING(s string) {
	p.inst(opSTRING, p.addString(s))
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
	p.inst(opJMPF, uint64(skip))
	if nt == parse.NodeWith {
		p.inst(opPUSHDOT, 0)
	}
	p.buf = append(p.buf, list...)
	if nt == parse.NodeWith {
		p.inst(opPOPDOT, 0)
	}
	if c := len(elseList); c > 0 {
		p.inst(opJMP, uint64(c))
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
		val := uint64(0)
		if x.True {
			val = 1
		}
		p.inst(opBOOL, val)
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
			p.inst(opFIELD, p.addString(v))
		}
	case *parse.IdentifierNode:
		if len(p.cmd) == 0 {
			return fmt.Errorf("identifier %q outside of command?", x.Ident)
		}
		args := p.cmd[len(p.cmd)-1] - 1 // first ar is identifier
		val := uint64(args<<32) | p.addString(x.Ident)
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
			p.inst(opFLOAT, math.Float64bits(x.Float64))
		case x.IsInt:
			p.inst(opINT, uint64(x.Int64))
		case x.IsUint:
			p.inst(opUINT, x.Uint64)
		}
	case *parse.PipeNode:
		for _, v := range x.Cmds {
			if err := p.walk(v); err != nil {
				return err
			}
		}
		// TODO: Set variables
	case *parse.StringNode:
		p.addSTRING(x.Text)
	case *parse.TextNode:
		p.addWB(x.Text)
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
		return reflect.ValueOf(v.Interface())
	}
	return v
}
