package m6800

import (
    "fmt"
//    "log"
    "time"

    "github.com/bartgrantham/fpemu/mem"
    "github.com/bartgrantham/fpemu/pia"
//    "github.com/bartgrantham/fpemu/ui"
)

//var pia_buf uint8

type M6800 struct {
    PC      uint16
    X       uint16
    A       uint8
    B       uint8
    CC      uint8
    SP      uint16
    PIA     pia.PIA
    NMI     bool
//    cps     float64
//    cycles  float64
}

const (
    C   uint8  = 1 << iota  // carry
    V   // overflow
    Z   // zero
    N   // negative / sign
    I   // interrupt masked, set by hardware interrupt or SEI opcode
    H   // half-carry
)

func NewM6800(mmu mem.MMU16, pia pia.PIA) *M6800{
    m := M6800{PC:mmu.R16(0xFFFE), PIA:pia}
//    m.SEI_0f(mmu)  // CPU starts with interrupts disabled/masked
    return &m
}

func (m *M6800) Status() string {
    fmtstr := "PC: $%.4X ($%.4X) ; X:$%.4X ; A:0x%.2X ; B:0x%.2X ; CC:0x%08b ; SP:$%.4X"
    return fmt.Sprintf(fmtstr, m.PC, m.PC, m.X, m.A, m.B, m.CC, m.SP)
}

func (m *M6800) Step(mmu mem.MMU16) error {
    mmu.ClearPeekCounts()
//    if m.NMI || (m.IRQ && (m.CC & I == 0)) {
//    ui.Log(fmt.Sprintf("PC: $%.4x  IRQ0:%v IRQ1:%v intmask:%v", m.PC, m.PIA.IRQ(0), m.PIA.IRQ(1), m.CC & I != I))
//    ui.Log(fmt.Sprintf("%v", time.Now()))
    if (m.PIA.IRQ(0) || m.PIA.IRQ(1)) && (m.CC & I != I) {
        m.save_registers(mmu)
        m.SEI_0f(mmu)  // interrupts masked
        if m.NMI {
            m.PC = mmu.R16(0xFFFC)
        } else {
            m.PC = mmu.R16(0xFFF8)
        }
    }
    opcode := mmu.R8(m.PC)
    m.PC += 1
    return m.dispatch(opcode, mmu)
}

/*
* When RTI is executed the saved registers are restored and processing proceeds from the interrupted point
* On IRQ:
    * If I=0 and the IRQ line goes low for at least one ϕ2 cycle
    * registers CC, B, A, X, PC (7 bytes, in that order, big endian for X and PC) are stored at SP-6..SP
    * I is set to 1 (IRQs masked)
    * 16-bit (big endian) irq vector is loaded from $FFF8 and the irq begins processing
* On NMI:
    * If the NMI line goes low for at least one ϕ2 cycle
    * registers CC, B, A, X, PC (7 bytes, in that order, big endian for X and PC) are stored at SP-6..SP
    * I is set to 1 (IRQs masked)
    * 16-bit (big endian) irq vector is loaded from $FFFC and the irq begins processing
* On SWI:
    * registers CC, B, A, X, PC (7 bytes, in that order, big endian for X and PC) are stored at SP-6..SP
    * I is set to 1 (IRQs masked)
    * 16-bit (big endian) irq vector is loaded from $FFFA and the irq begins processing

On firepower port A is the DAC, port B is from the mainboard
*/


func (m *M6800) Run(mmu mem.MMU16, ctrl chan rune) {
    var chr rune
    var tick *time.Ticker
    tick = time.NewTicker(time.Millisecond)
    _ = tick
    go func() {
        for {
            select {
                case chr = <-ctrl:
                    if chr >= 'a' && chr <= 'p' {
                        m.PIA.Write(1, uint8(chr-'a'))
                    }
                //case <-tick.C:
                default:
                    // run 1ms worth of cycles
                    m.Step(mmu)
            }
        }
    }()
}
