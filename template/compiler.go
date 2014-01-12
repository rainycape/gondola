package template

import (
	"fmt"
	"gnd.la/util/internal/templateutil"
	"gnd.la/util/types"
	"html/template"
	"io"
	"io/ioutil"
	"reflect"
	"runtime"
	"strings"
	"text/template/parse"
)

var (
	errType          = reflect.TypeOf((*error)(nil)).Elem()
	stringType       = reflect.TypeOf("")
	htmlType         = reflect.TypeOf(HTML(""))
	templateHtmlType = reflect.TypeOf(template.HTML(""))
	stringerType     = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	emptyType        = reflect.TypeOf((*interface{})(nil)).Elem()
	zero             = reflect.Zero(emptyType)
)

// TODO: Remove variables inside if or with when exiting the scope

type opcode uint8

const (
	opNOP opcode = iota
	opFIELD
	opFUNC
	opITER
	opJMP
	opJMPE
	opJMPF
	opJMPT
	opMARK
	opNEXT
	opDOT
	opPRINT
	opPUSHDOT
	opPOPDOT
	opPOP
	opSETVAR
	opSTRING
	opTEMPLATE
	opVAL
	opVAR
	opWB
)

type valType uint32

type inst struct {
	op  opcode
	val valType
}

func encodeVal(high int, low valType) valType {
	return valType(high<<16) | low
}

func decodeVal(val valType) (int, int) {
	return int(val >> 16), int(val & 0xFFFF)
}

var endIter = reflect.ValueOf(-1)

type iterator interface {
	Next() (reflect.Value, reflect.Value)
}

type nilIterator struct {
}

func (it *nilIterator) Next() (reflect.Value, reflect.Value) {
	return endIter, reflect.Value{}
}

type sliceIterator struct {
	val    reflect.Value
	pos    int
	length int
}

func (it *sliceIterator) Next() (reflect.Value, reflect.Value) {
	if it.pos < it.length {
		val := it.val.Index(it.pos)
		pos := reflect.ValueOf(it.pos)
		it.pos++
		return pos, val
	}
	return endIter, reflect.Value{}
}

type mapIterator struct {
	val  reflect.Value
	keys []reflect.Value
	pos  int
}

func (it *mapIterator) Next() (reflect.Value, reflect.Value) {
	if it.pos < len(it.keys) {
		k := it.keys[it.pos]
		val := it.val.MapIndex(k)
		it.pos++
		return k, val
	}
	return endIter, reflect.Value{}
}

type chanIterator struct {
	val reflect.Value
	pos int
}

func (it *chanIterator) Next() (reflect.Value, reflect.Value) {
	pos := endIter
	val, ok := it.val.Recv()
	if ok {
		pos = reflect.ValueOf(it.pos)
		it.pos++
	}
	return pos, val
}

func newIterator(v reflect.Value) (iterator, error) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return &nilIterator{}, nil
		}
	}
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			return &nilIterator{}, nil
		}
		return &sliceIterator{val: v, length: v.Len()}, nil
	case reflect.Map:
		if v.Len() == 0 {
			return &nilIterator{}, nil
		}
		keys := v.MapKeys()
		types.SortValues(keys)
		return &mapIterator{val: v, keys: keys}, nil
	case reflect.Chan:
		if v.IsNil() {
			return &nilIterator{}, nil
		}
		return &chanIterator{val: v}, nil
	case reflect.Invalid:
		return &nilIterator{}, nil
	}
	return nil, fmt.Errorf("can't range over %T", v.Interface())
}

type variable struct {
	name  string
	value reflect.Value
}

var statePool = make(chan *state, runtime.GOMAXPROCS(0))

type state struct {
	p     *Program
	w     io.Writer
	vars  []variable
	stack []reflect.Value
	marks []int
	dot   []reflect.Value
}

func newState(p *Program, w io.Writer) *state {
	select {
	case s := <-statePool:
		s.reset()
		s.p = p
		s.w = w
		return s
	default:
		return &state{
			p: p,
			w: w,
		}
	}
}

func (s *state) dup() *state {
	ns := newState(s.p, s.w)
	ns.vars = append(ns.vars, s.vars[0]) // pass $Vars
	return ns
}

func (s *state) reset() {
	s.vars = s.vars[:0]
	s.stack = s.stack[:0]
	s.marks = s.marks[:0]
	s.dot = s.dot[:0]
}

func (s *state) put() {
	select {
	case statePool <- s:
	default:
	}
}

func (s *state) formatErr(pc int, tmpl string, err error) error {
	tr := s.p.tmpl.Trees[tmpl]
	if tr != nil {
		ctx := s.p.context[tmpl]
		for ii, v := range ctx {
			if v.pc > pc {
				if ii > 0 {
					node := ctx[ii-1].node
					loc, _ := tr.ErrorContext(node)
					if loc != "" {
						// Need to check if the error was in line 1, to remove
						// the length of the text we prepend to templates.
						// TODO: Actually, that's prepended after every define,
						// eventually fix that too.
						p := strings.SplitN(loc, ":", 2)
						if len(p) == 2 {
							var line int
							var pos int
							if _, err := fmt.Sscanf(p[1], "%d:%d", &line, &pos); err == nil {
								if line == 1 {
									pos -= len(templatePrepend)
								}
								loc = fmt.Sprintf("%s:%d:%d", p[0], line, pos)
							}
						}
						return fmt.Errorf("%s: %s", loc, err.Error())
					}
				}
				break
			}
		}
	}
	return err
}

func (s *state) errorf(pc int, tmpl string, format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	return s.formatErr(pc, tmpl, err)
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

func (s *state) varValue(name string) (reflect.Value, error) {
	for i := s.varMark() - 1; i >= 0; i-- {
		if s.vars[i].name == name {
			return s.vars[i].value, nil
		}
	}
	return reflect.Value{}, fmt.Errorf("undefined variable: %q", name)
}

// call fn, remove its args from the stack and push the result
func (s *state) call(fn reflect.Value, name string, args int) error {
	//fmt.Println("WILL CALL", name, args, len(s.stack), s.stack)
	pos := len(s.stack) - args
	in := s.stack[pos:]
	// arguments are in reverse order
	for ii := 0; ii < len(in)/2; ii++ {
		in[ii], in[len(in)-1-ii] = in[len(in)-1-ii], in[ii]
	}
	res := fn.Call(in)
	//fmt.Println("CALLED", name, res)
	if len(res) == 2 && !res[1].IsNil() {
		return fmt.Errorf("error calling %q: %s", name, res[1].Interface())
	}
	s.stack = append(s.stack[:pos], stackable(res[0]))
	return nil
}

func (s *state) execute(tmpl string, dot reflect.Value) error {
	code := s.p.code[tmpl]
	if code == nil {
		return fmt.Errorf("template %q does not exist", tmpl)
	}
	s.dot = []reflect.Value{dot}
	s.pushVar("", dot)
	for pc := 0; pc < len(code); pc++ {
		v := code[pc]
		switch v.op {
		case opMARK:
			s.marks = append(s.marks, len(s.stack))
		case opPOP:
			if v.val == 0 {
				// if and else blocks pop before entering the block, so we might have no mark
				if len(s.marks) > 0 {
					// POP until mark
					p := len(s.marks) - 1
					s.stack = s.stack[:s.marks[p]]
					s.marks = s.marks[:p]
				}
			} else {
				s.stack = s.stack[:len(s.stack)-int(v.val)]
			}
		case opFIELD:
			res := zero
			p := len(s.stack) - 1
			top := s.stack[p]
			args, i := decodeVal(v.val)
			if top.IsValid() {
				if top.Kind() == reflect.Map && top.Type().Key().Kind() == reflect.String {
					k := s.p.rstrings[i]
					res = stackable(top.MapIndex(k))
					/*if !res.IsValid() {
						res = reflect.Zero(top.Type().Elem())
					}*/
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
						if err := s.call(fn, name, args); err != nil {
							return err
						}
						// s.call already puts the result in the stack
						break
					}
					// try to get a field by that name
					for top.Kind() == reflect.Ptr {
						if top.IsNil() {
							return s.errorf(pc, tmpl, "nil pointer evaluationg field %q on type %T", name, top.Interface())
						}
						top = top.Elem()
					}
					if top.Kind() != reflect.Struct {
						return s.errorf(pc, tmpl, "can't evaluate field on type %T", top.Interface())
					}
					res = top.FieldByName(name)
					if !res.IsValid() {
						// TODO: Check if the type has a pointer method which we couldn't
						// address, to provide a better error message
						return s.errorf(pc, tmpl, "%q is not a field of struct type %T", name, top.Interface())
					}
				}
			}
			// opFIELD overwrites the stack
			s.stack[p] = res
		case opFUNC:
			args, i := decodeVal(v.val)
			// function existence is checked at compile time
			fn := s.p.funcs[i]
			if err := s.call(fn.val, fn.name, args); err != nil {
				return s.formatErr(pc, tmpl, err)
			}
		case opVAR:
			name := s.p.strings[int(v.val)]
			v, err := s.varValue(name)
			if err != nil {
				return s.formatErr(pc, tmpl, err)
			}
			s.stack = append(s.stack, v)
		case opDOT:
			s.stack = append(s.stack, dot)
		case opITER:
			iter, err := newIterator(s.stack[len(s.stack)-1])
			if err != nil {
				return s.formatErr(pc, tmpl, err)
			}
			s.stack = append(s.stack, reflect.ValueOf(iter))
		case opNEXT:
			iter, ok := s.stack[len(s.stack)-1].Interface().(iterator)
			if !ok {
				return s.errorf(pc, tmpl, "NEXT called without iterator")
			}
			idx, val := iter.Next()
			s.stack = append(s.stack, idx, val)
		case opJMP:
			pc += int(int32(v.val))
		case opJMPF:
			p := len(s.stack)
			if p == 0 || !isTrue(s.stack[p-1]) {
				pc += int(int32(v.val))
			}
		case opJMPE:
			p := len(s.stack) - 2
			idx := s.stack[p]
			if idx.Kind() == reflect.Int && idx.Int() == -1 {
				// pop idx and val
				s.stack = s.stack[:p]
				pc += int(int32(v.val))
			}
		case opSETVAR:
			name := s.p.strings[int(v.val)]
			p := len(s.stack) - 1
			s.pushVar(name, s.stack[p])
		case opTEMPLATE:
			name := s.p.strings[int(v.val)]
			dup := s.dup()
			dupDot := s.stack[len(s.stack)-1]
			err := dup.execute(name, dupDot)
			dup.put()
			if err != nil {
				// execute already returns the formatted error
				return err
			}
		case opVAL:
			s.stack = append(s.stack, s.p.values[v.val])
		case opJMPT:
			p := len(s.stack)
			if p > 0 && isTrue(s.stack[p-1]) {
				pc += int(int32(v.val))
			}
		case opPRINT:
			v := s.stack[len(s.stack)-1]
			val, ok := printableValue(v)
			if !ok {
				return s.errorf(pc, tmpl, "can't print value of type %s", v.Type())
			}
			if _, err := fmt.Fprint(s.w, val); err != nil {
				return s.formatErr(pc, tmpl, err)
			}
		case opPUSHDOT:
			s.dot = append(s.dot, dot)
			dot = s.stack[len(s.stack)-1]
		case opPOPDOT:
			p := len(s.dot) - 1
			dot = s.dot[p]
			s.dot = s.dot[:p]
		case opSTRING:
			s.stack = append(s.stack, s.p.rstrings[int(v.val)])
		case opWB:
			if _, err := s.w.Write(s.p.bs[int(v.val)]); err != nil {
				return s.formatErr(pc, tmpl, err)
			}
		default:
			return s.errorf(pc, tmpl, "invalid opcode %d", v.op)
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

type context struct {
	pc   int
	node parse.Node
}

type scratch struct {
	buf        []inst
	cmd        []int
	pipe       []int
	ctx        []context
	noPrint    bool
	branchPipe bool
}

// snap returns a *scratch with the buf amd ctx of this
// scratch, while setting this scratch's buf and
// ctx to nil. used while compiling branches
func (s *scratch) snap() *scratch {
	ret := &scratch{buf: s.buf, ctx: s.ctx}
	s.buf = nil
	s.ctx = nil
	return ret
}

// restore sets the scrach buf and ctx to p.buf and p.ctx
func (s *scratch) restore(p *scratch) {
	s.buf = p.buf
	s.ctx = p.ctx
}

// prepend prepends one instruction to the scrach, adjusting
// the values in ctx
func (s *scratch) prepend(op opcode, val valType) *scratch {
	s.buf = append([]inst{{op: op, val: val}}, s.buf...)
	for _, v := range s.ctx {
		v.pc++
	}
	return s
}

func (s *scratch) popFront(count int) *scratch {
	s.buf = s.buf[count:]
	for _, v := range s.ctx {
		v.pc -= count
	}
	return s
}

func (s *scratch) append(op opcode, val valType) *scratch {
	s.buf = append(s.buf, inst{op: op, val: val})
	return s
}

// add adds another scratch at the end of this one. ctx in p
// is adjusted.
func (s *scratch) add(p *scratch) {
	for _, v := range s.ctx {
		v.pc += len(s.buf)
	}
	s.buf = append(s.buf, p.buf...)
	s.ctx = append(s.ctx, s.ctx...)
}

type Program struct {
	tmpl     *Template
	funcs    []*fn
	strings  []string
	rstrings []reflect.Value
	values   []reflect.Value
	bs       [][]byte
	code     map[string][]inst
	context  map[string][]context
	// used only during compilation
	s *scratch
}

func (p *Program) inst(op opcode, val valType) {
	p.s.buf = append(p.s.buf, inst{op: op, val: val})
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
	if len(p.s.cmd) > 0 {
		args = p.s.cmd[len(p.s.cmd)-1] - 1 // first arg is the FieldNode
		if len(p.s.pipe) > 0 && p.s.pipe[len(p.s.pipe)-1] > 0 {
			args++
		}
	}
	p.inst(opFIELD, encodeVal(args, p.addString(name)))
}

func (p *Program) prevFuncReturnType() reflect.Type {
	if len(p.s.buf) > 0 {
		if in := p.s.buf[len(p.s.buf)-1]; in.op == opFUNC {
			_, i := decodeVal(in.val)
			fn := p.funcs[i]
			return fn.val.Type().Out(0)
		}
	}
	return nil
}

func (p *Program) walkBranch(nt parse.NodeType, b *parse.BranchNode) error {
	p.s.branchPipe = true
	if err := p.walk(b.Pipe); err != nil {
		return err
	}
	p.s.branchPipe = false
	saved := p.s.snap()
	if err := p.walk(b.List); err != nil {
		return err
	}
	list := p.s.snap().prepend(opPOP, 0)
	var elseList *scratch
	if b.ElseList != nil {
		if err := p.walk(b.ElseList); err != nil {
			return err
		}
		// if the else is empty, just ignore it
		if len(p.s.buf) > 0 {
			elseList = p.s.snap().prepend(opPOP, 0)
		}
	}
	p.s.restore(saved)
	skip := len(list.buf)
	if elseList != nil {
		// Skip the JMP at the start of the elseList
		skip += 1
	}
	switch nt {
	case parse.NodeIf:
		p.inst(opJMPF, valType(skip))
		p.s.add(list)
	case parse.NodeWith:
		// if false, skip the PUSHDOT and POPDOT
		p.inst(opJMPF, valType(skip+2))
		p.inst(opPUSHDOT, 0)
		p.s.add(list)
		p.inst(opPOPDOT, 0)
	case parse.NodeRange:
		// remove the opPOP from the list, we need
		// iter to be kept on the stack. add also PUSHDOT
		// and POPDOT.
		list.popFront(1).prepend(opPUSHDOT, 0).append(opPOPDOT, 0)
		toPop := 2
		// if there are variables declared, add instructions
		// for setting them
		if len(b.Pipe.Decl) > 0 {
			toPop--
			list.prepend(opPOP, 1).prepend(opSETVAR, p.addString(b.Pipe.Decl[0].Ident[0][1:]))
			if len(b.Pipe.Decl) > 1 {
				toPop--
				list.prepend(opPOP, 1).prepend(opSETVAR, p.addString(b.Pipe.Decl[1].Ident[0][1:]))
			}
		}
		// pop variables which haven't beeen popped yet
		if toPop > 0 {
			list.append(opPOP, valType(toPop))
		}
		// add a jump back to 2 instructions before the
		// list, which will call NEXT and JMPE again.
		list.append(opJMP, valType(-len(list.buf)-3))
		// initialize the iter
		p.inst(opITER, 0)
		// call next for the first time
		p.inst(opNEXT, 0)
		if elseList == nil {
			// no elseList. just iterate and jump out of the
			// loop once we reach the end of the iteration
			p.inst(opJMPE, valType(len(list.buf)))
		} else {
			// if the iteration stopped in the first step, we
			// need to jump to elseList, skipping the JMP at its
			// start (for range loops the JMP is not really needed,
			// but one extra instruction won't hurt much). We also
			// need to skip the 3 instructions following this one.
			p.inst(opJMPE, valType(len(list.buf)+1+3))
			// Now jump the following two instructions, they're used for
			// subsequent iterations
			p.inst(opJMP, 2)
			// 2nd and the rest of iterations start here
			p.inst(opNEXT, 0)
			// If ended, jump outside list and elseList
			out := len(list.buf) + 1
			if elseList != nil {
				out += len(elseList.buf)
			}
			p.inst(opJMPE, valType(out))
		}
		p.s.add(list)
	default:
		return fmt.Errorf("invalid branch type %v", nt)
	}
	if elseList != nil {
		p.inst(opJMP, valType(len(elseList.buf)))
		p.s.add(elseList)
	}
	return nil
}

func (p *Program) walk(n parse.Node) error {
	switch x := n.(type) {
	case *parse.ActionNode:
		p.inst(opMARK, 0)
		if err := p.walk(x.Pipe); err != nil {
			return err
		}
		if len(x.Pipe.Decl) == 0 {
			if p.s.noPrint {
				p.s.noPrint = false
			} else {
				p.inst(opPRINT, 0)
			}
		}
		p.inst(opPOP, 0)
	case *parse.BoolNode:
		p.inst(opVAL, p.addValue(x.True))
	case *parse.CommandNode:
		p.s.cmd = append(p.s.cmd, len(x.Args))
		// Command nodes are pushed on reverse order, so they are
		// evaluated from right to left. If we encounter a function
		// while executing it, we can just grab the arguments from the stack
		for ii := len(x.Args) - 1; ii >= 0; ii-- {
			node := x.Args[ii]
			if err := p.walk(node); err != nil {
				return err
			}
		}
		p.s.cmd = p.s.cmd[:len(p.s.cmd)-1]
	case *parse.DotNode:
		p.inst(opDOT, 0)
	case *parse.FieldNode:
		p.inst(opDOT, 0)
		for _, v := range x.Ident {
			p.addFIELD(v)
		}
	case *parse.IdentifierNode:
		if len(p.s.cmd) == 0 {
			return fmt.Errorf("identifier %q outside of command?", x.Ident)
		}
		if x.Ident == topAssetsFuncName || x.Ident == bottomAssetsFuncName {
			// These functions won't change their output once we reach this
			// point and they return template.HTML. We can just call them now
			// and add a WB instruction with the result, thus avoiding the function
			// call and the useless escaping. We also need to ignore the rest
			// of the pipeline.
			var b []byte
			if x.Ident == topAssetsFuncName {
				b = []byte(p.tmpl.topAssets)
			} else {
				b = []byte(p.tmpl.bottomAssets)
			}
			p.addWB(b)
			p.s.noPrint = true
			break
		}
		name := x.Ident
		if p.s.noPrint && strings.HasPrefix(name, "html_") {
			break
		}
		if name == "html_template_htmlescaper" {
			// Check if the input of this function is a string or template.HTML
			// and either use the specialized function or remove the escaping
			// entirely when possible.
			if typ := p.prevFuncReturnType(); typ != nil {
				switch {
				case types.IsNumeric(typ):
					name = ""
				case typ.Kind() == reflect.String:
					switch typ {
					case stringType:
						name = "html_template_htmlstringescaper"
					case htmlType, templateHtmlType:
						name = ""
					}
				}
			}
		}
		if name == "" {
			// Function optimized away
			break
		}
		args := p.s.cmd[len(p.s.cmd)-1] - 1 // first arg is identifier
		if len(p.s.pipe) > 0 && p.s.pipe[len(p.s.pipe)-1] > 0 {
			args++
		}
		fn := p.tmpl.funcMap[name]
		if fn == nil {
			return fmt.Errorf("undefined function %q", name)
		}
		p.inst(opFUNC, encodeVal(args, p.addFunc(fn, name)))
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
		var val valType
		switch {
		case x.IsComplex:
			val = p.addValue(x.Complex128)
		case x.IsFloat:
			val = p.addValue(x.Float64)
		case x.IsInt:
			val = p.addValue(x.Int64)
		case x.IsUint:
			val = p.addValue(x.Uint64)
		default:
			return fmt.Errorf("invalid number node %v", x)
		}
		p.inst(opVAL, val)
	case *parse.PipeNode:
		for ii, v := range x.Cmds {
			p.s.pipe = append(p.s.pipe, ii)
			if err := p.walk(v); err != nil {
				return err
			}
			p.s.pipe = p.s.pipe[:len(p.s.pipe)-1]
		}
		if !p.s.branchPipe {
			for _, variable := range x.Decl {
				// Remove $
				p.inst(opSETVAR, p.addString(variable.Ident[0][1:]))
			}
		}
	case *parse.RangeNode:
		if err := p.walkBranch(parse.NodeRange, &x.BranchNode); err != nil {
			return err
		}
	case *parse.StringNode:
		p.addSTRING(x.Text)
	case *parse.TemplateNode:
		if err := p.walk(x.Pipe); err != nil {
			return err
		}
		p.inst(opTEMPLATE, p.addString(x.Name))
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
	p.s.ctx = append(p.s.ctx, context{pc: len(p.s.buf), node: n})
	return nil
}

func (p *Program) Execute(w io.Writer, data interface{}) error {
	return p.ExecuteTemplateVars(w, p.tmpl.Root(), data, nil)
}

func (p *Program) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	return p.ExecuteTemplateVars(w, name, data, nil)
}

func (p *Program) ExecuteVars(w io.Writer, data interface{}, vars VarMap) error {
	return p.ExecuteTemplateVars(w, "", data, vars)
}

func (p *Program) ExecuteTemplateVars(w io.Writer, name string, data interface{}, vars VarMap) error {
	s := newState(p, w)
	s.pushVar("Vars", reflect.ValueOf(vars))
	err := s.execute(name, reflect.ValueOf(data))
	s.put()
	return err
}

func NewProgram(tmpl *Template) (*Program, error) {
	// Need to execute it once, for html/template to add
	// the escaping hooks.
	tmpl.Execute(ioutil.Discard, nil)
	// Add escaping functions
	tmpl.Funcs(htmlEscapeFuncs)
	p := &Program{tmpl: tmpl, code: make(map[string][]inst), context: make(map[string][]context)}
	for k, v := range tmpl.Trees {
		root := simplifyList(v.Root)
		p.s = new(scratch)
		if err := p.walk(root); err != nil {
			return nil, err
		}
		p.code[k] = p.s.buf
		p.context[k] = p.s.ctx
		p.s = nil
	}
	return p, nil
}

// simplifyList removes all nodes injected by Gondola
// to implement the Vars system, since *Program implements
// support for that directly and does not require introducing
// extra nodes.
func simplifyList(root *parse.ListNode) *parse.ListNode {
	if len(root.Nodes) > 0 && root.Nodes[0].String() == fmt.Sprintf("{{$%s := .%s}}", varsKey, varsKey) {
		count := len(root.Nodes)
		if wn, ok := root.Nodes[count-1].(*parse.WithNode); ok && wn.Pipe.String() == "."+dataKey {
			list := wn.List
			list.Nodes = append(root.Nodes[1:count-1], wn.List.Nodes...)
			var templates []*parse.TemplateNode
			templateutil.WalkNode(wn.List, nil, func(n, p parse.Node) {
				if tn, ok := n.(*parse.TemplateNode); ok {
					templates = append(templates, tn)
				}
			})
			for _, v := range templates {
				if v.Pipe != nil {
					if len(v.Pipe.Cmds) == 1 {
						cmd := v.Pipe.Cmds[0]
						switch len(cmd.Args) {
						case 3: // template had no arguments
							v.Pipe = nil
						case 5: // original pipe was 5th arg
							v.Pipe = cmd.Args[4].(*parse.PipeNode)
						default:
							fmt.Println(len(cmd.Args), v.Pipe)
							panic("something went bad")
						}
					}
				}
			}
			return list
		}
	}
	return root
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
