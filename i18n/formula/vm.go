package formula

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
)

// This virtual machine is able to execute most formulas
// much faster than the AST walking interpreter, but
// it's more restricted. By default, a formula will be
// first compiled using the VM. If that fails, it will
// fall back to AST walking.
//
// The VM is quite simple, having only a general purpose
// register (R) and a boolean status register (S).
// Some instructions might contain an integer value (V):
//
//  N - set R = n
//  MOD - set R = R % V
//  JMPT - jump by V if S is true
//  JMPF - jump by V if S is false
//  EQ - set S = (R == V)
//  NEQ - set S = (R != V)
//  LT - set S = (R < V)
//  LTE - set S = (R <= V)
//  GT - set S = (R > V)
//  GTE - set S = (R >= V)
//  RET - end execution and return V
//
// If the end of the program is reached without finding
// a ret instruction, the last value of S is returned.
// as an integer.

type opCode int

const (
	// Instructions altering R
	opN opCode = iota + 1
	opMOD
	// Special instructions
	opRET
	// Jump instructions
	opJMPT
	opJMPF
	// Comparison instructions
	opEQ
	opNEQ
	opLT
	opLTE
	opGT
	opGTE
)

func (o opCode) String() string {
	names := []string{"N", "MOD", "RET", "JMPT", "JMPF", "EQ", "NEQ", "LT", "LTE", "GT", "GTE"}
	return names[int(o)-1]
}

func (o opCode) Alters() bool {
	return o <= opMOD
}

func (o opCode) IsSpecial() bool {
	return o == opRET
}

func (o opCode) IsJump() bool {
	return o == opJMPT || o == opJMPF
}

func (o opCode) Compares() bool {
	return o >= opEQ
}

type instruction struct {
	opCode opCode
	value  int
}

func invalid(s *scanner.Scanner, what, val string) (Formula, error) {
	return nil, fmt.Errorf("invalid %s in formula at %s: %q", what, s.Pos(), val)
}

func jumpTarget(s *scanner.Scanner, form string, chr byte) int {
	// look for matching :
	offset := s.Pos().Offset
	paren := 0
	target := -1
	for ii, v := range []byte(form[offset:]) {
		if v == '(' {
			paren++
		} else if v == ')' {
			paren--
			if paren < 0 {
				target = offset + ii
				break
			}
		} else if v == chr && paren == 0 {
			target = offset + ii
			break
		}
	}
	return target
}

func makeJump(s *scanner.Scanner, form string, code *[]*instruction, op opCode, jumps map[int][]*instruction, chr byte) {
	// end of conditional, put the placeholder for a jump
	// and complete it once we reach the matching chr. Store the
	// current position of the jump in its value, so
	// calculating the relative offset is quicker.
	pos := len(*code)
	inst := &instruction{opCode: op, value: pos}
	*code = append(*code, inst)
	target := jumpTarget(s, form, chr)
	jumps[target] = append(jumps[target], inst)
}

func resolveJumps(s *scanner.Scanner, code []*instruction, jumps map[int][]*instruction) {
	// check for incomplete jumps to this location.
	// the pc should point at the next instruction
	// to be added and the jump is relative.
	pc := len(code)
	offset := s.Pos().Offset - 1
	for _, v := range jumps[offset] {
		v.value = pc - v.value - 1
	}
	delete(jumps, offset)
}

func compileVmFormula(form string) (Formula, error) {
	var s scanner.Scanner
	var err error
	s.Init(strings.NewReader(form))
	s.Error = func(s *scanner.Scanner, msg string) {
		err = fmt.Errorf("error parsing plural formula %s: %s", s.Pos(), msg)
	}
	s.Mode = scanner.ScanIdents | scanner.ScanInts
	tok := s.Scan()
	var code []*instruction
	var op bytes.Buffer
	var logic bytes.Buffer
	jumps := make(map[int][]*instruction)
	for tok != scanner.EOF && err == nil {
		switch tok {
		case scanner.Ident:
			if n := s.TokenText(); n != "n" {
				return invalid(&s, "ident", n)
			}
			code = append(code, &instruction{opCode: opN})
		case scanner.Int:
			val, _ := strconv.Atoi(s.TokenText())
			if op.Len() == 0 {
				// return statement
				code = append(code, &instruction{opCode: opRET, value: val})
			} else {
				var opc opCode
				switch op.String() {
				case "%":
					opc = opMOD
				case "==":
					opc = opEQ
				case "!=":
					opc = opNEQ
				case "<":
					opc = opLT
				case "<=":
					opc = opLTE
				case ">":
					opc = opGT
				case ">=":
					opc = opGTE
				default:
					return invalid(&s, "op", op.String())
				}
				code = append(code, &instruction{opCode: opc, value: val})
				op.Reset()
			}
		case '?':
			resolveJumps(&s, code, jumps)
			makeJump(&s, form, &code, opJMPF, jumps, ':')
		case ':':
			resolveJumps(&s, code, jumps)
		case '!', '=', '<', '>', '%':
			op.WriteRune(tok)
		case '&', '|':
			// logic operations
			if logic.Len() == 0 {
				logic.WriteRune(tok)
			} else if logic.Len() == 1 {
				b := logic.Bytes()[0]
				if b != byte(tok) {
					return invalid(&s, "token", string(tok))
				}
				if b == '&' {
					makeJump(&s, form, &code, opJMPF, jumps, ':')
				} else {
					makeJump(&s, form, &code, opJMPT, jumps, '?')
				}
				logic.Reset()
			} else {
				return invalid(&s, "token", string(tok))
			}
		case '(':
		case ')':
			resolveJumps(&s, code, jumps)
		default:
			return invalid(&s, "token", string(tok))
		}
		tok = s.Scan()
	}
	//	fmt.Println("BC for", form)
	//	for ii, v := range code {
	//		fmt.Printf("%d:%s\t%d\n", ii, v.opCode, v.value)
	//	}
	return makeVmFunc(code), nil
}

func makeVmFunc(insts []*instruction) Formula {
	count := len(insts)
	return func(n int) int {
		return vmExec(insts, count, n)
	}
}

func vmExec(insts []*instruction, count int, n int) int {
	var R int
	var S bool
	for ii := 0; ii < count; ii++ {
		i := insts[ii]
		switch i.opCode {
		case opN:
			R = n
		case opMOD:
			R = R % i.value
		case opRET:
			return i.value
		case opJMPT:
			if S {
				ii += i.value
			}
		case opJMPF:
			if !S {
				ii += i.value
			}
		case opEQ:
			S = R == i.value
		case opNEQ:
			S = R != i.value
		case opLT:
			S = R < i.value
		case opLTE:
			S = R <= i.value
		case opGT:
			S = R > i.value
		case opGTE:
			S = R >= i.value
		}
	}
	return bint(S)
}
