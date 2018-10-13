package m6800

import (
    "fmt"

    "github.com/bartgrantham/fpemu/mem"
//    "github.com/bartgrantham/fpemu/ui"
)

/*
From the 1975 manual:

2-1.2
* H is set when there's a carry from b3 to b4
* I is set by hardware interrupts or SEI opcode
* N is set if b7 is set
* Z is set if the result is all zeros
* C is set if there is a carry from b7
* V is set if there was an arithmetic overflow
    * overflow of A+B=R is defined as: V = (A:7 == B:7) & (R:7 != A:7)  (A and B same sign, R opposite)
    * overflow of A-B=R is defined as: V = (A:7 == !B:7) & (R:7 != A:7) (A and -B same sign, R opposite)

* REI restores the previous interrupt flag from the stack by restoring CC
* note that calling CLI at the beginning of an interrupt allows a higher priority interrupt to occur, because the registers will be saved if another interrupt happens.  The manual explicitly mentions this nesting.

*/


// TODO: turn this into a table w/ cycle counts, etc.
func (m *M6800) dispatch(opcode uint8, mmu mem.MMU16) error {
    switch opcode {
        case 0x08: m.INX_08(mmu)
        case 0x0e: m.CLI_0e(mmu)
        case 0x0f: m.SEI_0f(mmu)
        case 0x20: m.BRA_20(mmu)
        case 0x26: m.BNE_26(mmu)
        case 0x27: m.BEQ_27(mmu)
        case 0x32: m.PUL_32(mmu)
//        case 0x33: m.PUL_33(mmu)  // untested
        case 0x36: m.PSH_36(mmu)
//        case 0x37: m.PSH_37(mmu)  // untested
        case 0x39: m.RTS_39(mmu)
        case 0x43: m.COM_43(mmu)
        case 0x4c: m.INC_4c(mmu)
        case 0x4f: m.CLR_4f(mmu)
        case 0x53: m.COM_53(mmu)
        case 0x5a: m.DEC_5a(mmu)
        case 0x6f: m.CLR_6f(mmu)
        case 0x7c: m.INC_7c(mmu)
        case 0x7d: m.TST_7d(mmu)
        case 0x7e: m.JMP_7e(mmu)
        case 0x7f: m.CLR_7f(mmu)
        case 0x81: m.CMP_81(mmu)
        case 0x84: m.AND_84(mmu)
        case 0x85: m.BIT_85(mmu)
        case 0x86: m.LDA_86(mmu)
        case 0x89: m.ADC_89(mmu)
        case 0x8e: m.LDS_8e(mmu)
        case 0x96: m.LDA_96(mmu)
        case 0x97: m.STA_97(mmu)
        case 0x9b: m.ADD_9b(mmu)
        case 0xa0: m.SUB_a0(mmu)
        case 0xa6: m.LDA_a6(mmu)
        case 0xa7: m.STA_a7(mmu)
        case 0xb6: m.LDA_b6(mmu)
        case 0xbd: m.JSR_bd(mmu)
        case 0xc1: m.CMP_c1(mmu)
        case 0xc6: m.LDA_c6(mmu)
        case 0xce: m.LDX_ce(mmu)
        case 0xd6: m.LDA_d6(mmu)
        case 0xd7: m.STA_d7(mmu)
        case 0xde: m.LDX_de(mmu)
        case 0xdf: m.STX_df(mmu)
        case 0xe6: m.LDA_e6(mmu)
        case 0xe7: m.STA_e7(mmu)
        case 0xef: m.STX_ef(mmu)
        case 0xf6: m.LDA_f6(mmu)
        case 0xf7: m.STA_f7(mmu)
        default:
            m.unimplmented(opcode, mmu)
    }
    return nil
}



// Catch-all opcode
func (m *M6800) unimplmented(opcode uint8, mmu mem.MMU16) {
    status := fmt.Sprintf("\nUnimplmented opcode: %.2X\n    CPU status: %s", opcode, m.Status())
    panic(status)
}

func (m *M6800) INX_08(mmu mem.MMU16) {
    m.X += 1
    if m.X == 0 {
        m.CC |= Z
    } else {
        m.CC &= Z
    }
}

// CLI, clear interrupt flag (enables inerrupts)
func (m *M6800) CLI_0e(mmu mem.MMU16) {
    m.CC &= ^I
}

// SEI, set interrupt flag (disables interrupts)
func (m *M6800) SEI_0f(mmu mem.MMU16) {
    m.CC |= I
}

func (m *M6800) BRA_20(mmu mem.MMU16) {
    offset := int32(int8(mmu.R8(m.PC))) + 2  // range -126..128
    m.PC -= 1
    // uint16(int32) will truncate properly (ie. no conversion), negative number handled properly
    m.PC += uint16(offset)
}

// maybe software int?
func (m*M6800) save_registers(mmu mem.MMU16) {
    mmu.W8(m.SP-0, m.CC)
    mmu.W8(m.SP-1, m.B)
    mmu.W8(m.SP-2, m.A)
    mmu.W16(m.SP-4, m.X)
    mmu.W16(m.SP-6, m.PC)
    m.SP -= 7
}

func (m *M6800) BNE_26(mmu mem.MMU16) {
    if m.CC & Z != Z {
        offset := mmu.R8(m.PC)
        m.PC += 1
        if offset & 0x80 == 0x80 {
            offset = (offset ^ 0xff)+1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    } else {
        m.PC += 1
    }
}

func (m *M6800) BEQ_27(mmu mem.MMU16) {
    if m.CC & Z == Z {
        m.PC += uint16(mmu.R8(m.PC))
    }
    m.PC += 1
}

func (m *M6800) PUL_32(mmu mem.MMU16) {
    m.SP += 1
    m.A = mmu.R8(m.SP)
}

func (m *M6800) PUL_33(mmu mem.MMU16) {
    m.SP += 1
    m.B = mmu.R8(m.SP)
}

func (m *M6800) PSH_36(mmu mem.MMU16) {
    mmu.W8(m.SP, m.A)
    m.SP -= 1
}

func (m *M6800) PSH_37(mmu mem.MMU16) {
    mmu.W8(m.SP, m.B)
    m.SP -= 1
}

func (m *M6800) RTS_39(mmu mem.MMU16) {
    m.PC = mmu.R16(m.SP+1)
    m.SP -= 2
}

func (m *M6800) RTI_3b(mmu mem.MMU16) {
    //saved registers are restored and processing proceeds
    panic("not yet")
}

func (m *M6800) COM_43(mmu mem.MMU16) {
    m.A ^= 0xff
    m.CC |= C
    m.CC &= ^V
    m.set_NZ8(m.A)
}

func (m *M6800) INC_4c(mmu mem.MMU16) {
    m.A += 1
    if m.A == 0x00 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.set_NZ8(m.A)
    m.PC += 2
}

func (m *M6800) CLR_4f(mmu mem.MMU16) {
    m.A = 0
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
}

func (m *M6800) COM_53(mmu mem.MMU16) {
    m.B ^= 0xff
    m.CC |= C
    m.CC &= ^V
    m.set_NZ8(m.B)
}

func (m *M6800) DEC_5a(mmu mem.MMU16) {
    if m.B == 0x80 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.B -= 1
    m.set_NZ8(m.B)
}

func (m *M6800) CLR_6f(mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), 0)
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
    m.PC += 1
}

func (m *M6800) INC_7c(mmu mem.MMU16) {
    tmp := mmu.R8(mmu.R16(m.PC)) + 1
    mmu.W8(mmu.R16(m.PC), tmp)
    if tmp == 0 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.set_NZ8(tmp)
    m.PC += 2
}

func (m *M6800) TST_7d(mmu mem.MMU16) {
    tmp := mmu.R8(m.PC)
    m.CC &= ^C
    m.CC &= ^V
    m.set_NZ8(tmp)
    m.PC += 2
}

func (m *M6800) JMP_7e(mmu mem.MMU16) {
    m.PC = mmu.R16(m.PC)
    m.PC += 2
}

func (m *M6800) CLR_7f(mmu mem.MMU16) {
    mmu.W8(mmu.R16(m.PC), 0)
    m.A = 0
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
    m.PC += 2
}

func (m *M6800) CMP_81(mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(m.PC)
/*
    if minuend < subtrahend {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    difference := minuend + (^subtrahend + 1)
    if cmp_V(minuend, subtrahend, difference) {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.set_NZ8(difference)
*/
    _ = m.sub(minuend, subtrahend)
    m.PC += 1
}

func (m *M6800) AND_84(mmu mem.MMU16) {
    m.A &= mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func (m *M6800) BIT_85(mmu mem.MMU16) {
    m.CC &= ^V
    m.set_NZ8(m.A & mmu.R8(m.PC))
    m.PC += 1
}

func (m *M6800) LDA_86(mmu mem.MMU16) {
    m.A = mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func (m *M6800) ADC_89(mmu mem.MMU16) {
    augend := m.A
    addend := mmu.R8(uint16(mmu.R8(m.PC)))
    var sum uint8
    if m.CC & C == 0 {
        sum = augend + addend
    } else {
        sum = augend + addend + 1
    }
    if (augend & 0x80) != (sum & 0x80) {
       m.CC |= C
    } else {
       m.CC &= ^C
    }
    if add_V(augend, addend, sum) {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    if sum & 0x0f > 0x09 {
        m.CC |= H
    } else {
        m.CC &= ^H
    }
    m.set_NZ8(sum)
    m.A = sum
    m.PC += 1
}

func (m *M6800) LDS_8e(mmu mem.MMU16) {
    m.SP = mmu.R16(m.PC)
    m.CC &= ^V
    m.set_NZ16(m.SP)
    m.PC += 2
}

func (m *M6800) LDA_96(mmu mem.MMU16) {
    m.A = mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func (m *M6800) STA_97(mmu mem.MMU16) {
    mmu.W8(uint16(mmu.R8(m.PC)), m.A)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func (m *M6800) ADD_9b(mmu mem.MMU16) {
    augend := m.A
    addend := mmu.R8(uint16(mmu.R8(m.PC)))
    sum := augend + addend
    if (augend & 0x80) != (sum & 0x80) {
       m.CC |= C
    } else {
       m.CC &= ^C
    }
    if add_V(augend, addend, sum) {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    if sum & 0x0f > 0x09 {
        m.CC |= H
    } else {
        m.CC &= ^H
    }
    m.set_NZ8(sum)
    m.A = sum
    m.PC += 1
}

func (m *M6800) SUB_a0(mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
/*
    if minuend < subtrahend {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    difference := minuend + (^subtrahend + 1)
    if cmp_V(minuend, subtrahend, difference) {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.set_NZ8(difference)
*/
    m.A = m.sub(minuend, subtrahend)
    m.PC += 1
}

func (m *M6800) LDA_a6(mmu mem.MMU16) {
    m.A = mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^V  // clear overflow
    m.set_NZ8(m.A)
    m.PC += 1
}

func (m *M6800) STA_a7(mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), m.A)
    m.PC += 1
}

func (m *M6800) LDA_b6(mmu mem.MMU16) {
    m.A = mmu.R8(mmu.R16(m.PC))
    m.CC &= ^V  // clear overflow
    m.set_NZ8(m.A)
    m.PC += 2
}

func (m *M6800) JSR_bd(mmu mem.MMU16) {
    mmu.W16(m.SP-1, m.PC+2)
    m.SP -= 2
    m.PC = mmu.R16(m.PC)
}

func (m *M6800) CMP_c1(mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(m.PC)
/*
    if minuend < subtrahend {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    difference := minuend + (^subtrahend + 1)
    if cmp_V(minuend, subtrahend, difference) {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.set_NZ8(difference)
*/
    _ = m.sub(minuend, subtrahend)
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

func (m *M6800) LDA_d6(mmu mem.MMU16) {
    m.B = mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func (m *M6800) STA_d7(mmu mem.MMU16) {
    mmu.W8(uint16(mmu.R8(m.PC)), m.B)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func (m *M6800) LDX_de(mmu mem.MMU16) {
    m.X = mmu.R16(uint16(mmu.R8(m.PC)))
    m.PC += 1
}

func (m *M6800) STX_df(mmu mem.MMU16) {
    mmu.W16(uint16(mmu.R8(m.PC)), m.X)
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 1
}

func (m *M6800) LDA_e6(mmu mem.MMU16) {
    m.B = mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^V  // clear overflow
    m.set_NZ8(m.B)
    m.PC += 1
}

func (m *M6800) STA_e7(mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), m.B)
    m.PC += 1
}

func (m *M6800) STX_ef(mmu mem.MMU16) {
    //mmu.W16(uint16(mmu.R8(m.PC)), m.X)
    mmu.W16(m.X + uint16(mmu.R8(m.PC)), m.X)
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 1
}

func (m *M6800) LDA_f6(mmu mem.MMU16) {
    m.B = mmu.R8(m.PC)
    m.CC &= ^V  // clear overflow
    m.set_NZ8(m.B)
    m.PC += 2
}

func (m *M6800) STA_f7(mmu mem.MMU16) {
    mmu.W8(mmu.R16(m.PC), m.B)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 2
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

func add_V(augend, addend, sum uint8) bool {
    ausign := augend & 0x80 == 0x80
    adsign := addend & 0x80 == 0x80
    ssign  := sum & 0x80 == 0x80
    // false == positive, true == negative!
    switch {
        case !ausign && !adsign && ssign:
            // positive + positive == negative
            return true
        case ausign && adsign && !ssign:
            // negative + negative == positive
            return true
    }
    return false
}

// common function to all SUB and CMP opcodes, handles flags but caller handles advancing PC
func (m *M6800) sub(minuend, subtrahend uint8) uint8 {
    difference := minuend + (^subtrahend + 1)

    // carry
    if minuend < subtrahend {
        m.CC |= C
    } else {
        m.CC &= ^C
    }

    // overflow
    msign := minuend & 0x80 == 0x80
    ssign := subtrahend & 0x80 == 0x80
    dsign := difference & 0x80 == 0x80
    // false == positive, true == negative!
    switch {
        case !msign && ssign && dsign:
            // positive - negative == negative
            m.CC |= V
        case msign && !ssign && !dsign:
            // negative - positive == positive
            m.CC |= V
        default:
            m.CC &= ^V
    }
    m.set_NZ8(difference)
    return difference
}

/*
func cmp_V(minuend, subtrahend, difference uint8) bool {
    msign := minuend & 0x80 == 0x80
    ssign := subtrahend & 0x80 == 0x80
    dsign := difference & 0x80 == 0x80
    // false == positive, true == negative!
    switch {
        case !msign && ssign && dsign:
            // positive - negative == negative
            return true
        case msign && !ssign && !dsign:
            // negative - positive == positive
            return true
    }
    return false
}
*/
