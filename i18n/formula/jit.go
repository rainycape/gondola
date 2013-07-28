package formula

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

var (
	errJitNotSupported = errors.New("formula JIT is not supported on this os/arch")
	errEmptyProgram    = errors.New("can't JIT an empty program")
	asm                assembler
	mmap               mmapper
)

type assembler interface {
	WriteOpN(w io.Writer) error
	WriteCompare(w io.Writer, n int32) error
	WriteMod(w io.Writer, n int32) error
	WriteInt32(w io.Writer, n int32) error
	// cmp is the opcode for the comparison just before this
	// this jump. If the jump branchs if false, this function
	// will receive the inverse comparison.
	WriteJump(w io.Writer, cmp opCode) error
	WriteReturn(w io.Writer, n int32) error
}

type mmapper interface {
	Map(code []byte) (Formula, error)
}

func jump(buf *bytes.Buffer, jumps *[]int, targets map[int]int, jump, cmp *instruction) error {
	opCode := cmp.opCode
	if jump.opCode == opJMPF {
		opCode = opCode.Inverse()
	}
	if err := asm.WriteJump(buf, opCode); err != nil {
		return err
	}
	// put a jump of 0 as a placeholder
	if err := asm.WriteInt32(buf, 0); err != nil {
		return err
	}
	j := buf.Len()
	*jumps = append(*jumps, j)
	targets[j] = jump.value
	return nil
}

func vmJit(p program) (f Formula, err error) {
	if asm == nil || mmap == nil {
		return nil, errJitNotSupported
	}
	if len(p) == 0 {
		return nil, errEmptyProgram
	}
	var buf bytes.Buffer
	var ends []int
	var jumps []int
	targets := make(map[int]int)
	if err = asm.WriteOpN(&buf); err != nil {
		return
	}
	for ii, v := range p {
		switch v.opCode {
		case opN:
			err = asm.WriteOpN(&buf)
		case opMOD, opNMOD:
			if v.opCode == opNMOD && ii > 0 {
				if err = asm.WriteOpN(&buf); err != nil {
					return
				}
			}
			err = asm.WriteMod(&buf, int32(v.value))
		case opJMPT, opJMPF:
			err = jump(&buf, &jumps, targets, p[ii], p[ii-1])
		case opEQ, opNEQ, opLT, opLTE, opGT, opGTE:
			err = asm.WriteCompare(&buf, int32(v.value))
		case opRET:
			err = asm.WriteReturn(&buf, int32(v.value))
		default:
			return nil, fmt.Errorf("can't map VM instruction %d", v.opCode)
		}
		ends = append(ends, buf.Len())
	}
	if err != nil {
		return
	}
	if last := p[len(p)-1]; last.opCode != opRET {
		j := &instruction{opJMPF, 1}
		if err = jump(&buf, &jumps, targets, j, last); err != nil {
			return
		}
		ends = append(ends, buf.Len())
		if err = asm.WriteReturn(&buf, 1); err != nil {
			return
		}
		ends = append(ends, buf.Len())
		if err = asm.WriteReturn(&buf, 0); err != nil {
			return
		}
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
		skip := targets[v]
		offset := ends[j+skip] - ends[j]
		var b bytes.Buffer
		if err = asm.WriteInt32(&b, int32(offset)); err != nil {
			return
		}
		n := b.Bytes()
		for ii := 0; ii < 4; ii++ {
			code[v-(4-ii)] = n[ii]
		}
	}
	/*fmt.Println("JIT")
	for _, v := range code {
		fmt.Printf("%02x ", v)
	}
	fmt.Println("")*/
	return mmap.Map(code)
}
