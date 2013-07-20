package formula

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
)

// This VM is quite simple, having only a general purpose
// register (R) and a boolean status register (S).
// Some instructions might contain an integer value (V):
//
//  N - set R = n
//  ADD - set R = R + V
//  SUB - set R = R - V
//  MULT - set R = R * V
//  DIV - set R = R / V
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

type opCode uint8

const (
	// Instructions altering R
	opN opCode = iota + 1
	opADD
	opSUB
	opMULT
	opDIV
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
	names := []string{"N", "ADD", "SUB", "MULT", "DIV", "MOD", "RET", "JMPT", "JMPF", "EQ", "NEQ", "LT", "LTE", "GT", "GTE"}
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

func invalid(s *scanner.Scanner, what, val string) ([]*instruction, error) {
	return nil, fmt.Errorf("invalid %s in formula at %s: %q", what, s.Pos(), val)
}

func jumpTarget(s *scanner.Scanner, form string, chr byte) int {
	// look for matching chr
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
	code, err := vmCompile(form)
	if err != nil {
		return nil, err
	}
	code = vmOptimize(code)
	return makeVmFunc(code), nil
}

func vmCompile(form string) ([]*instruction, error) {
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
			if op.Len() > 0 {
				return invalid(&s, "ident", "RHS variables are not supported")
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
				case "+":
					opc = opADD
				case "-":
					opc = opSUB
				case "*":
					opc = opMULT
				case "/":
					opc = opDIV
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
	return code, nil
}

func removeInstructions(insts []*instruction, start int, count int) []*instruction {
	insts = append(insts[:start], insts[start+count:]...)
	// Check for jumps that might be affected by the removal
	for kk := start; kk >= 0; kk-- {
		if in := insts[kk]; in.opCode.IsJump() && kk+in.value > start {
			in.value -= count
		}
	}
	return insts
}

func vmOptimize(insts []*instruction) []*instruction {
	// The optimizer is quite simple. Each pass is documented
	// at its beginning.

	//A first pass looks
	// for multiple comparison instructions that are preceeded
	// by exactly the same instructions and it removes the second
	// group of instructions.
	cmp := -1
	count := len(insts)
	ii := 0
	for ; ii < count; ii++ {
		v := insts[ii]
		if v.opCode.Compares() {
			if cmp >= 0 {
				delta := ii - cmp
				jj := cmp - 1
				for ; jj >= 0; jj-- {
					i1 := insts[jj]
					if !i1.opCode.Alters() {
						break
					}
					i2 := insts[jj+delta]
					if i1.opCode != i2.opCode || i1.value != i2.value {
						break
					}
				}
				equal := (cmp - 1) - jj
				if equal > 0 {
					ii -= equal
					count -= equal
					insts = removeInstructions(insts, ii, equal)
					continue
				}
			}
			cmp = ii
		}
	}
	// A second pass then looks for
	// instructions that set R = N when R is already
	// equal to N and it removes the second instruction.
	n := -1
	for ii = 0; ii < count; ii++ {
		v := insts[ii]
		if v.opCode == opN {
			if n >= 0 {
				insts = removeInstructions(insts, ii, 1)
				ii--
				count--
				continue
			}
			n = ii
		} else if v.opCode.Alters() {
			n = -1
		}
	}
	// Third pass looks for jumps which end up in a jump of the same type,
	// add adjusts the value to make just one jump.
	for ii := 0; ii < count; ii++ {
		v := insts[ii]
		if v.opCode.IsJump() {
			for true {
				t := ii + v.value + 1
				if t >= count {
					break
				}
				nv := insts[t]
				if nv.opCode != v.opCode {
					break
				}
				v.value += nv.value
			}
		}
	}
	return insts
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
		case opADD:
			R += i.value
		case opSUB:
			R -= i.value
		case opMULT:
			R *= i.value
		case opDIV:
			R /= i.value
		case opMOD:
			R %= i.value
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
