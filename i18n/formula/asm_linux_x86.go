// +build !appengine
// +build 386 amd64

package formula

import (
	"encoding/binary"
	"fmt"
	"io"
)

type x86Assembler struct {
}

func (a *x86Assembler) WriteOpN(w io.Writer) error {
	// mov 0x8(%esp) to %ebx
	_, err := w.Write([]byte{0x8b, 0x5c, 0x24, 0x08})
	return err
}

func (a *x86Assembler) WriteCompare(w io.Writer, n int32) error {
	// cmp %ebx, im32
	if _, err := w.Write([]byte{0x81, 0xfb}); err != nil {
		return err
	}
	return a.WriteInt32(w, n)
}

func (a *x86Assembler) WriteMod(w io.Writer, n int32) error {
	// mov %ebx to %eax
	if _, err := w.Write([]byte{0x89, 0xd8}); err != nil {
		return err
	}
	// cqto
	if _, err := w.Write([]byte{0x99}); err != nil {
		return err
	}
	// mov divisor to %ebx
	if _, err := w.Write([]byte{0xc7, 0xc3}); err != nil {
		return err
	}
	if err := a.WriteInt32(w, n); err != nil {
		return err
	}
	// idiv %ebx
	if _, err := w.Write([]byte{0xf7, 0xfb}); err != nil {
		return err
	}
	// mov %rdx to %ebx
	if _, err := w.Write([]byte{0x89, 0xd3}); err != nil {
		return err
	}
	return nil
}

func (a *x86Assembler) WriteInt32(w io.Writer, n int32) error {
	return binary.Write(w, binary.LittleEndian, n)
}

func (a *x86Assembler) WriteJump(w io.Writer, cmp opCode) error {
	var b byte
	switch cmp {
	case opEQ:
		b = 0x84
	case opNEQ:
		b = 0x85
	case opLT:
		b = 0x8C
	case opLTE:
		b = 0x8E
	case opGT:
		b = 0x8F
	case opGTE:
		b = 0x8D
	default:
		return fmt.Errorf("can't jump depending on opcode %d", cmp)
	}
	_, err := w.Write([]byte{0x0f, b})
	return err
}

func (a *x86Assembler) WriteReturn(w io.Writer, n int32) error {
	// mov imm32 0x10(%esp) to %ebx
	if _, err := w.Write([]byte{0xc7, 0x44, 0x24, 0x10}); err != nil {
		return err
	}
	if err := a.WriteInt32(w, n); err != nil {
		return err
	}
	// ret
	_, err := w.Write([]byte{0xc3})
	return err
}

func init() {
	asm = &x86Assembler{}
}
