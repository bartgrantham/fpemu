package m6800

import (
    "fmt"

    "github.com/bartgrantham/fpemu/mem"
)

// TODO: turn this into a table w/ cycle counts, etc.
func (m *M6800) dispatch(opcode uint8, mmu mem.MMU16) error {
    switch opcode {
        case 0x0e: m.CLI_0e(mmu)
        case 0x0f: m.SEI_0f(mmu)
        case 0x20: m.BRA_20(mmu)
        case 0x4f: m.CLR_4f(mmu)
        case 0x6f: m.CLR_6f(mmu)
        case 0x86: m.LDA_86(mmu)
        case 0x8e: m.LDS_8e(mmu)
        case 0x97: m.STA_97(mmu)
        case 0xa7: m.STA_a7(mmu)
        case 0xc6: m.LDA_c6(mmu)
        case 0xce: m.LDX_ce(mmu)
        case 0xe7: m.STA_e7(mmu)
        default:
            m.unimplmented(opcode, mmu)
    }
    return nil
}


// flags: HI NZVC
//        21 8421

func (m *M6800) set_NZ8(val uint8) {
    if val == 0 {
        m.CC |= Z  // set Z
    } else {
        m.CC &= ^Z // clear Z
    }
    if val & 0x80 == 0x80 {
        m.CC |= N
    } else {
        m.CC &= ^N
    }
}

func (m *M6800) set_NZ16(val uint16) {
    if val == 0 {
        m.CC |= Z  // set Z
    } else {
        m.CC &= ^Z // clear Z
    }
    if val & 0x8000 == 0x8000 {
        m.CC |= N
    } else {
        m.CC &= ^N
    }
}

// CLI, clear interrupt flag (enables inerrupts)
func (m *M6800) CLI_0e(mmu mem.MMU16) {
    m.CC &= 0xfe
}

// SEI, set interrupt flag (disables interrupts)
func (m *M6800) SEI_0f(mmu mem.MMU16) {
    m.CC |= 0x01
}

func (m *M6800) BRA_20(mmu mem.MMU16) {
    offset := int32(int8(mmu.R8(m.PC))) + 2  // range -126..128
    m.PC -= 1
    // uint16(int32) will truncate properly (ie. no conversion), negative number handled properly
    m.PC += uint16(offset)
}

func (m *M6800) CLR_4f(mmu mem.MMU16) {
    m.A = 0
    m.CC |= 0x4   // set Z
    m.CC &= 0xF4  // clear NVC
}

func (m *M6800) CLR_6f(mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), 0)
    m.CC |= 0x4   // set Z
    m.CC &= 0xF4  // clear NVC
    m.PC += 1
}

func (m *M6800) LDA_86(mmu mem.MMU16) {
    m.A = mmu.R8(m.PC)
    m.set_NZ8(m.A)
    m.PC += 1
}

func (m *M6800) LDS_8e(mmu mem.MMU16) {
    m.SP = mmu.R16(m.PC)
    m.CC &= ^V  // clear overflow
    m.set_NZ16(m.SP)
    m.PC += 2
}

func (m *M6800) STA_97(mmu mem.MMU16) {
    mmu.W8(uint16(mmu.R8(m.PC)), m.A)
    m.PC += 1
}

func (m *M6800) STA_a7(mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), m.A)
    m.PC += 1
}

func (m *M6800) LDA_c6(mmu mem.MMU16) {
    m.B = mmu.R8(m.PC)
    m.PC += 1
}

func (m *M6800) LDX_ce(mmu mem.MMU16) {
    m.X = mmu.R16(m.PC)
    m.PC += 2
}

func (m *M6800) STA_e7(mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), m.B)
    m.PC += 1
}

func (m *M6800) unimplmented(opcode uint8, mmu mem.MMU16) {
    status := fmt.Sprintf("\nUnimplmented opcode: %.2X\n    CPU status: %s", opcode, m.Status())
    panic(status)
}

