// +build !appengine,amd64,linux

package formula

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func intToQw(n int) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int32(n))
	return buf.Bytes()
}

func jump(buf *bytes.Buffer, jumps *[]int, jump, cmp *instruction) error {
	opCode := cmp.opCode
	if jump.opCode == opJMPF {
		opCode = opCode.Inverse()
	}
	var b byte
	switch opCode {
	case opEQ:
		b = 0x74
	case opNEQ:
		b = 0x75
	case opLT:
		b = 0x7C
	case opLTE:
		b = 0x7E
	case opGT:
		b = 0x7F
	case opGTE:
		b = 0x7D
	default:
		return fmt.Errorf("can't jump depending on opcode %d", opCode)
	}
	buf.WriteByte(b)
	// TODO: Fail if cmp.value > 255
	buf.WriteByte(byte(jump.value))
	*jumps = append(*jumps, buf.Len())
	return nil
}

func ret(buf *bytes.Buffer, value int) {
	movq := []byte{0x48, 0xc7, 0x44, 0x24, 0x10}
	retc := []byte{0xc3}
	buf.Write(movq)
	buf.Write(intToQw(value))
	buf.Write(retc)
}

func vmJit(p program) (Formula, error) {
	if len(p) == 0 {
		return nil, errEmptyProgram
	}
	var buf bytes.Buffer
	opn := []byte{0x48, 0x8b, 0x5c, 0x24, 0x08}
	cmp := []byte{0x48, 0x83, 0xfb}
	var ends []int
	var jumps []int
	// Start every program by setting %rbx to n
	buf.Write(opn)
	for ii, v := range p {
		switch v.opCode {
		case opN:
			buf.Write(opn)
		case opMOD, opNMOD:
			if v.opCode == opNMOD && ii > 0 {
				buf.Write(opn)
			}
			// mov %rbx to %rax
			buf.Write([]byte{0x48, 0x89, 0xd8})
			// cqto
			buf.Write([]byte{0x48, 0x99})
			// mov divisor to %rbx
			buf.Write([]byte{0x48, 0xc7, 0xc3})
			buf.Write(intToQw(v.value))
			// idiv %rbx
			buf.Write([]byte{0x48, 0xf7, 0xfb})
			// mov %rdx to %rbx
			buf.Write([]byte{0x48, 0x89, 0xd3})
		case opRET:
			ret(&buf, v.value)
		case opJMPT, opJMPF:
			if err := jump(&buf, &jumps, p[ii], p[ii-1]); err != nil {
				return nil, err
			}
		case opEQ, opNEQ, opLT, opLTE, opGT, opGTE:
			if v.value > 256 || v.value < 0 {
				return nil, fmt.Errorf("can't compare values > 256 or < 0")
			}
			buf.Write(cmp)
			buf.WriteByte(byte(v.value))
		default:
			return nil, fmt.Errorf("can't map VM instruction %d", v.opCode)
		}
		ends = append(ends, buf.Len())
	}
	if last := p[len(p)-1]; last.opCode != opRET {
		j := &instruction{opJMPF, 1}
		if err := jump(&buf, &jumps, j, last); err != nil {
			return nil, err
		}
		ends = append(ends, buf.Len())
		ret(&buf, 1)
		ends = append(ends, buf.Len())
		ret(&buf, 0)
		ends = append(ends, buf.Len())
	}
	code := buf.Bytes()
	for _, v := range jumps {
		j := -1
		for ii, val := range ends {
			if val == v {
				j = ii
				break
			}
		}
		if j == -1 {
			return nil, fmt.Errorf("can't map jump ending at %d", v)
		}
		offset := ends[j+int(code[v-1])] - ends[j]
		// TODO: Fail if offset > 255
		code[v-1] = byte(offset)
	}
	/*	fmt.Println("JIT")
		for _, v := range code {
			fmt.Printf("%02x ", v)
		}
		fmt.Println("")*/
	return makeJitFunc(code)
}
