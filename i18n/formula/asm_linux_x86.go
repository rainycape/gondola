// +build !appengine
// +build 386 amd64

package formula

import (
	"encoding/binary"
	"fmt"
	"io"
)

type x86Asm struct {
}

func (a *x86Asm) WriteOpN(w io.Writer) error {
	// mov 0x8(%esp) to %ebx
	_, err := w.Write([]byte{0x8b, 0x5c, 0x24, 0x08})
	return err
}

func (a *x86Asm) WriteCompare(w io.Writer, n int32) error {
	// cmp %ebx, im32
	if _, err := w.Write([]byte{0x81, 0xfb}); err != nil {
		return err
	}
	return a.WriteInt32(w, n)
}

func (a *x86Asm) WriteAdd(w io.Writer, n int32) error {
	// add %ebx, im32
	if _, err := w.Write([]byte{0x81, 0xc3}); err != nil {
		return err
	}
	return a.WriteInt32(w, n)
}

func (a *x86Asm) WriteSub(w io.Writer, n int32) error {
	// sub %ebx, im32
	if _, err := w.Write([]byte{0x81, 0xeb}); err != nil {
		return err
	}
	return a.WriteInt32(w, n)
}

func (a *x86Asm) WriteMult(w io.Writer, n int32) error {
	// imul %ebx, %ebx, im32
	if _, err := w.Write([]byte{0x69, 0xdb}); err != nil {
		return err
	}
	return a.WriteInt32(w, n)
}

func (a *x86Asm) writeDiv(w io.Writer, n int32, reg byte) error {
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
	// quotient is in %eax, remainder in %edx. reg
	// indicates which register to copy. It will be
	// either 0 (resulting on %eax) or 010000 (resulting
	// in %edx)
	if _, err := w.Write([]byte{0x89, 0xc3 | reg}); err != nil {
		return err
	}
	return nil
}

func (a *x86Asm) WriteDiv(w io.Writer, n int32) error {
	// grab %eax after idiv
	return a.writeDiv(w, n, 0)
}

func (a *x86Asm) WriteMod(w io.Writer, n int32) error {
	// grab %edx after idiv
	return a.writeDiv(w, n, 1<<4)
}

func (a *x86Asm) WriteJump(w io.Writer, cmp opCode) error {
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

func (a *x86Asm) WriteReturn(w io.Writer, n int32) error {
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

func (a *x86Asm) WriteInt32(w io.Writer, n int32) error {
	return binary.Write(w, binary.LittleEndian, n)
}

func init() {
	asm = &x86Asm{}
}
