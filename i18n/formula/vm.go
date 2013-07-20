package formula

import (
	"bytes"
	"fmt"
	"strconv"
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

type program []*instruction

func invalid(s *scanner.Scanner, what, val string) (program, error) {
	return nil, fmt.Errorf("invalid %s in formula at %s: %q", what, s.Pos(), val)
}

func jumpTarget(s *scanner.Scanner, code []byte, chr byte) int {
	// look for matching chr
	offset := s.Pos().Offset
	paren := 0
	target := -1
	for ii, v := range code[offset:] {
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

func makeJump(s *scanner.Scanner, code []byte, p *program, op opCode, jumps map[int][]int, chr byte) {
	// end of conditional, put the placeholder for a jump
	// and complete it once we reach the matching chr. Store the
	// current position of the jump in its value, so
	// calculating the relative offset is quicker.
	pos := len(*p)
	inst := &instruction{opCode: op, value: pos}
	*p = append(*p, inst)
	target := jumpTarget(s, code, chr)
	jumps[target] = append(jumps[target], pos)
}

func resolveJumps(s *scanner.Scanner, p program, jumps map[int][]int) {
	// check for incomplete jumps to this location.
	// the pc should point at the next instruction
	// to be added and the jump is relative.
	pc := len(p)
	offset := s.Pos().Offset - 1
	for _, v := range jumps[offset] {
		inst := p[v]
		inst.value = pc - inst.value - 1
	}
	delete(jumps, offset)
}

func compileVmFormula(code string) (Formula, error) {
	p, err := vmCompile(code)
	if err != nil {
		return nil, err
	}
	p = vmOptimize(p)
	return makeVmFunc(p), nil
}

func vmCompile(code string) (program, error) {
	var s scanner.Scanner
	var err error
	codeb := []byte(code)
	s.Init(bytes.NewReader(codeb))
	s.Error = func(s *scanner.Scanner, msg string) {
		err = fmt.Errorf("error parsing plural formula %s: %s", s.Pos(), msg)
	}
	s.Mode = scanner.ScanIdents | scanner.ScanInts
	tok := s.Scan()
	var p program
	var op bytes.Buffer
	var logic bytes.Buffer
	jumps := make(map[int][]int)
	for tok != scanner.EOF && err == nil {
		switch tok {
		case scanner.Ident:
			if n := s.TokenText(); n != "n" {
				return invalid(&s, "ident", n)
			}
			if op.Len() > 0 {
				return invalid(&s, "ident", "RHS variables are not supported")
			}
			p = append(p, &instruction{opCode: opN})
		case scanner.Int:
			val, _ := strconv.Atoi(s.TokenText())
			if op.Len() == 0 {
				// return statement
				p = append(p, &instruction{opCode: opRET, value: val})
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
				p = append(p, &instruction{opCode: opc, value: val})
				op.Reset()
			}
		case '?':
			resolveJumps(&s, p, jumps)
			makeJump(&s, codeb, &p, opJMPF, jumps, ':')
		case ':':
			resolveJumps(&s, p, jumps)
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
					makeJump(&s, codeb, &p, opJMPF, jumps, ':')
				} else {
					makeJump(&s, codeb, &p, opJMPT, jumps, '?')
				}
				logic.Reset()
			} else {
				return invalid(&s, "token", string(tok))
			}
		case '(':
		case ')':
			resolveJumps(&s, p, jumps)
		default:
			return invalid(&s, "token", string(tok))
		}
		tok = s.Scan()
	}
	return p, nil
}

func removeInstructions(p program, start int, count int) program {
	p = append(p[:start], p[start+count:]...)
	// Check for jumps that might be affected by the removal
	for kk := start; kk >= 0; kk-- {
		if in := p[kk]; in.opCode.IsJump() && kk+in.value > start {
			in.value -= count
		}
	}
	return p
}

func vmOptimize(insts program) program {
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

func makeVmFunc(p program) Formula {
	count := len(p)
	return func(n int) int {
		return vmExec(p, count, n)
	}
}

func vmExec(p program, count int, n int) int {
	var R int
	var S bool
	for ii := 0; ii < count; ii++ {
		i := p[ii]
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
