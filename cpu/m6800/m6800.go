package m6800

import (
    "fmt"
//    "log"

    "github.com/bartgrantham/fpemu/mem"
)

type M6800 struct {
    PC  uint16
    X   uint16
    A   uint8
    B   uint8
    CC  uint8
    SP  uint16
}

type Flag uint8
const (
    C   Flag  = 1 << iota  // carry
    V   // overflow
    Z   // zero
    N   // negative / sign
    I   // interrupt
    H   // half-carry
)

func NewM6800(mmu mem.MMU16) *M6800{
    m := M6800{PC:mmu.R16(0xFFFE)}
    return &m
}

func (m *M6800) Status() string {
    fmtstr := "PC: $%.4X ($%.4X) ; X:$%.4X ; A:0x%.2X ; B:0x%.2X ; CC:0x%08b ; SP:$%.4X"
    return fmt.Sprintf(fmtstr, m.PC & 0x7FF, m.PC, m.X, m.A, m.B, m.CC, m.SP)
}

func (m *M6800) Step(mmu mem.MMU16) error {
    opcode := mmu.R8(m.PC)
    m.PC += 1
    return m.dispatch(opcode, mmu)
}
