package m6800

import (
    "fmt"

    "github.com/bartgrantham/fpemu/mem"
//    "github.com/bartgrantham/fpemu/ui"
)

/*
* overflow of A+B=R is defined as: V = (A:7 == B:7) & (R:7 != A:7)  (A and B same sign, R opposite)
* overflow of A-B=R is defined as: V = (A:7 == !B:7) & (R:7 != A:7) (A and -B same sign, R opposite)

* REI restores the previous interrupt flag from the stack by restoring CC
* note that calling CLI at the beginning of an interrupt allows a higher priority interrupt to occur, because the registers will be saved if another interrupt happens.  The manual explicitly mentions this nesting.

*/

var cyclecounts [256]int = [256]int{
//  00  01  02  03  04  05  06  07  08  09  0A  0B  0C  0D  0E  0F
    0,  2,  0,  0,  0,  0,  2,  2,  4,  4,  2,  2,  2,  2,  2,  2,  // 00
    2,  2,  0,  0,  0,  0,  2,  2,  0,  2,  0,  2,  0,  0,  0,  0,  // 10
    4,  0,  4,  4,  4,  4,  4,  4,  4,  4,  4,  4,  4,  4,  4,  4,  // 20
    4,  4,  4,  4,  4,  4,  4,  4,  0,  5,  0,  10, 0,  0,  9,  12, // 30
    2,  0,  0,  2,  2,  0,  2,  2,  2,  2,  2,  0,  2,  2,  0,  2,  // 40
    2,  0,  0,  2,  2,  0,  2,  2,  2,  2,  2,  0,  2,  2,  0,  2,  // 50
    7,  0,  0,  7,  7,  0,  7,  7,  7,  7,  7,  0,  7,  7,  4,  7,  // 60
    6,  0,  0,  6,  6,  0,  6,  6,  6,  6,  6,  0,  6,  6,  3,  6,  // 70
    2,  2,  2,  0,  2,  2,  2,  0,  2,  2,  2,  2,  3,  8,  3,  0,  // 80
    3,  3,  3,  0,  3,  3,  3,  4,  3,  3,  3,  3,  4,  0,  4,  5,  // 90
    5,  5,  5,  0,  5,  5,  5,  6,  5,  5,  5,  5,  6,  8,  6,  7,  // A0
    4,  4,  4,  0,  4,  4,  4,  5,  4,  4,  4,  4,  5,  9,  5,  6,  // B0
    2,  2,  2,  0,  2,  2,  2,  0,  2,  2,  2,  2,  0,  0,  3,  0,  // C0
    3,  3,  3,  0,  3,  3,  3,  4,  3,  3,  3,  3,  0,  0,  4,  5,  // D0
    5,  5,  5,  0,  5,  5,  5,  6,  5,  5,  5,  5,  0,  0,  6,  7,  // E0
    4,  4,  4,  0,  4,  4,  4,  5,  4,  4,  4,  4,  0,  0,  5,  6,  // F0
}

var dispatch_table [256]func(*M6800, mem.MMU16) = [256]func(*M6800, mem.MMU16) {
//  00      01      02      03      04      05      06      07      08      09      0A      0B      0C      0D      0E      0F
    INVALD, NOP_01, INVALD, INVALD, INVALD, INVALD, UNIMPL, UNIMPL, INX_08, DEX_09, UNIMPL, UNIMPL, UNIMPL, UNIMPL, CLI_0e, SEI_0f, //00
    SBA_10, CMP_11, INVALD, INVALD, INVALD, INVALD, TAB_16, TBA_17, INVALD, UNIMPL, INVALD, ABA_1b, INVALD, INVALD, INVALD, INVALD, //10
    BRA_20, INVALD, BHI_22, BLS_23, BCC_24, BCS_25, BNE_26, BEQ_27, UNIMPL, UNIMPL, BPL_2a, BMI_2b, UNIMPL, UNIMPL, UNIMPL, UNIMPL, //20
    UNIMPL, UNIMPL, PUL_32, UNIMPL, UNIMPL, UNIMPL, PSH_36, UNIMPL, INVALD, RTS_39, INVALD, UNIMPL, INVALD, INVALD, UNIMPL, UNIMPL, //30
    UNIMPL, INVALD, INVALD, COM_43, LSR_44, INVALD, UNIMPL, UNIMPL, ASL_48, UNIMPL, DEC_4a, INVALD, INC_4c, TST_4d, INVALD, CLR_4f, //40
    UNIMPL, INVALD, INVALD, COM_53, LSR_54, INVALD, UNIMPL, UNIMPL, ASL_58, UNIMPL, DEC_5a, INVALD, INC_5c, TST_5d, INVALD, CLR_5f, //50
    UNIMPL, INVALD, INVALD, UNIMPL, UNIMPL, INVALD, UNIMPL, UNIMPL, UNIMPL, UNIMPL, UNIMPL, INVALD, UNIMPL, TST_6d, JMP_6e, CLR_6f, //60
    UNIMPL, INVALD, INVALD, COM_73, UNIMPL, INVALD, ROR_76, UNIMPL, UNIMPL, ROL_79, DEC_7a, INVALD, INC_7c, TST_7d, JMP_7e, CLR_7f, //70
    SUB_80, CMP_81, UNIMPL, INVALD, AND_84, BIT_85, LDA_86, INVALD, UNIMPL, ADC_89, ORA_8a, ADD_8b, CPX_8c, BSR_8d, LDS_8e, INVALD, //80
    UNIMPL, CMP_91, UNIMPL, INVALD, UNIMPL, UNIMPL, LDA_96, STA_97, EOR_98, UNIMPL, ORA_9a, ADD_9b, CPX_9c, INVALD, UNIMPL, UNIMPL, //90
    SUB_a0, UNIMPL, UNIMPL, INVALD, UNIMPL, UNIMPL, LDA_a6, STA_a7, UNIMPL, UNIMPL, UNIMPL, ADD_ab, UNIMPL, JSR_ad, UNIMPL, UNIMPL, //A0
    UNIMPL, UNIMPL, UNIMPL, INVALD, UNIMPL, UNIMPL, LDA_b6, STA_b7, UNIMPL, UNIMPL, UNIMPL, UNIMPL, UNIMPL, JSR_bd, UNIMPL, UNIMPL, //B0
    SUB_c0, CMP_c1, UNIMPL, INVALD, AND_c4, UNIMPL, LDA_c6, INVALD, UNIMPL, UNIMPL, UNIMPL, ADD_cb, INVALD, INVALD, LDX_ce, INVALD, //C0
    SUB_d0, UNIMPL, UNIMPL, INVALD, UNIMPL, UNIMPL, LDA_d6, STA_d7, UNIMPL, UNIMPL, UNIMPL, UNIMPL, INVALD, INVALD, LDX_de, STX_df, //D0
    UNIMPL, UNIMPL, UNIMPL, INVALD, UNIMPL, UNIMPL, LDA_e6, STA_e7, UNIMPL, UNIMPL, UNIMPL, UNIMPL, INVALD, INVALD, LDX_ee, STX_ef, //E0
    UNIMPL, UNIMPL, UNIMPL, INVALD, UNIMPL, UNIMPL, LDA_f6, STA_f7, UNIMPL, UNIMPL, UNIMPL, UNIMPL, INVALD, INVALD, LDX_fe, UNIMPL, //F0
//  00      01      02      03      04      05      06      07      08      09      0A      0B      0C      0D      0E      0F
}

var op uint8

func (m *M6800) dispatch(opcode uint8, mmu mem.MMU16) (int, error) {
    op = opcode
    dispatch_table[opcode](m, mmu)
    return cyclecounts[opcode], nil
}

func UNIMPL(m *M6800, mmu mem.MMU16) {
    status := fmt.Sprintf("\nUnimplmented opcode: %.2X\n    CPU status: %s", op, m.Status())
    panic(status)
}

func INVALD(m *M6800, mmu mem.MMU16) {
    status := fmt.Sprintf("\nInvalid opcode: %.2X\n    CPU status: %s", op, m.Status())
    panic(status)
}

// *
func NOP_01(m *M6800, mmu mem.MMU16) {
}

// *
func INX_08(m *M6800, mmu mem.MMU16) {
    m.X += 1
    if m.X == 0 {
        m.CC |= Z
    } else {
        m.CC &= Z
    }
}

// *
func DEX_09(m *M6800, mmu mem.MMU16) {
    m.X -= 1
    if m.X == 0 {
        m.CC |= Z
    } else {
        m.CC &= Z
    }
}

// *
func CLI_0e(m *M6800, mmu mem.MMU16) {
    m.CC &= ^I
}

// *
func SEI_0f(m *M6800, mmu mem.MMU16) {
    m.CC |= I
}

// *
func SBA_10(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := m.B
    m.A = m.sub(minuend, subtrahend, false)
}

// *
func CMP_11(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := m.B
    _ = m.sub(minuend, subtrahend, false)
}

// *
func TAB_16(m *M6800, mmu mem.MMU16) {
    m.B = m.A
    m.CC &= ^V
    m.set_NZ8(m.B)
}

// *
func TBA_17(m *M6800, mmu mem.MMU16) {
    m.A = m.B
    m.CC &= ^V
    m.set_NZ8(m.A)
}

// *
func ABA_1b(m *M6800, mmu mem.MMU16) {
    augend := m.A
    addend := m.B
    sum := m.add(augend, addend, false)
    m.A = sum
}

// *
func BRA_20(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    if offset & 0x80 == 0x80 {
        offset = ^offset + 1
        m.PC -= uint16(offset)
    } else {
        m.PC += uint16(offset)
    }
}

// *
func BHI_22(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    // no carry and the result wasn't zero
    if m.CC & (C | Z) == 0 {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// *
func BLS_23(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    // carry or the result was zero
    if (m.CC & C == C) || (m.CC & Z == Z) {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// *
func BCC_24(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    if m.CC & C != C {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// *
func BCS_25(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    if m.CC & C == C {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// *
func BNE_26(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    if m.CC & Z != Z {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// *
func BEQ_27(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    if m.CC & Z == Z {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// *
func BPL_2a(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    if m.CC & N != N {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// *
func BMI_2b(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    if m.CC & N == N {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// *
func PUL_32(m *M6800, mmu mem.MMU16) {
    m.SP += 1
    m.A = mmu.R8(m.SP)
}

// *
func PUL_33(m *M6800, mmu mem.MMU16) {
    m.SP += 1
    m.B = mmu.R8(m.SP)
}

// *
func PSH_36(m *M6800, mmu mem.MMU16) {
    mmu.W8(m.SP, m.A)
    m.SP -= 1
}

// *
func PSH_37(m *M6800, mmu mem.MMU16) {
    mmu.W8(m.SP, m.B)
    m.SP -= 1
}

// *
func RTS_39(m *M6800, mmu mem.MMU16) {
    m.PC = mmu.R16(m.SP+1)
    m.SP += 2
}

// *
func COM_43(m *M6800, mmu mem.MMU16) {
    m.A = ^m.A //0xff - m.A
    m.CC |= C
    m.CC &= ^V
    m.set_NZ8(m.A)
}

// *
func LSR_44(m *M6800, mmu mem.MMU16) {
    if m.A & 0x01 == 0x01 {
        m.CC |= C
        m.CC |= V
    }
    m.A >>= 1
    m.set_NZ8(m.A)
}

// *
func ASL_48(m *M6800, mmu mem.MMU16) {
    if m.A & 0x80 == 0x80 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    m.A = m.A << 1
    m.set_NZ8(m.A)
    switch {
        case (m.CC & N == N) && (m.CC & C != C):
            // negative, no carry
            m.CC |= V
        case (m.CC & N != N) && (m.CC & C == C):
            // positive, carry
            m.CC |= V
        default:
            m.CC &= ^V
    }
}

// *
func DEC_4a(m *M6800, mmu mem.MMU16) {
    if m.A == 0x80 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.A -= 1
    m.set_NZ8(m.A)
}

func INC_4c(m *M6800, mmu mem.MMU16) {
    if m.A == 0x7f {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.A += 1
    m.set_NZ8(m.A)
}

// *
func TST_4d(m *M6800, mmu mem.MMU16) {
    m.CC &= ^C
    m.CC &= ^V
    m.set_NZ8(m.A)
}

// *
func CLR_4f(m *M6800, mmu mem.MMU16) {
    m.A = 0
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
}

func COM_53(m *M6800, mmu mem.MMU16) {
    m.B = 0xff - m.B
    m.CC |= C
    m.CC &= ^V
    m.set_NZ8(m.B)
}

// *
func LSR_54(m *M6800, mmu mem.MMU16) {
    if m.B & 0x01 == 0x01 {
        m.CC |= C
        m.CC |= V
    } else {
        m.CC &= ^C
        m.CC &= ^V
    }
    m.B >>= 1
    m.set_NZ8(m.B)
}

// *
func ASL_58(m *M6800, mmu mem.MMU16) {
    if m.B & 0x80 == 0x80 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    m.B = m.B << 1
    m.set_NZ8(m.B)
    switch {
        case (m.CC & N == N) && (m.CC & C != C):
            // negative, no carry
            m.CC |= V
        case (m.CC & N != N) && (m.CC & C == C):
            // positive, carry
            m.CC |= V
        default:
            m.CC &= ^V
    }
}

// *
func DEC_5a(m *M6800, mmu mem.MMU16) {
    if m.B == 0x80 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.B -= 1
    m.set_NZ8(m.B)
}

// *
func INC_5c(m *M6800, mmu mem.MMU16) {
    if m.B == 0x7f {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.B += 1
    m.set_NZ8(m.B)
}

// *
func TST_5d(m *M6800, mmu mem.MMU16) {
    m.CC &= ^C
    m.CC &= ^V
    m.set_NZ8(m.B)
}

// *
func CLR_5f(m *M6800, mmu mem.MMU16) {
    m.B = 0
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
}

func TST_6d(m *M6800, mmu mem.MMU16) {
    tmp := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^C
    m.CC &= ^V
    m.set_NZ8(tmp)
    m.PC += 1
}

func JMP_6e(m *M6800, mmu mem.MMU16) {
    m.PC = m.X + uint16(mmu.R8(m.PC))
}

// *
func CLR_6f(m *M6800, mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), 0)
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
    m.PC += 1
}

// *
func COM_73(m *M6800, mmu mem.MMU16) {
    addr := mmu.R16(m.PC)
    tmp := ^mmu.R8(addr)
    m.CC |= C
    m.CC &= ^V
    m.set_NZ8(tmp)
    mmu.W8(addr, tmp)
    m.PC += 2
}

// *
func ROR_76(m *M6800, mmu mem.MMU16) {
    addr := mmu.R16(m.PC)
    tmp := mmu.R8(addr)
    wrap := m.CC & C == C
    if tmp & 0x01 == 0x01 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    tmp = tmp >> 1
    if wrap {
        tmp |= 0x80
    }
    m.set_NZ8(tmp)
    switch {
        case (m.CC & N == N) && (m.CC & C != C):
            // negative, no carry
            m.CC |= V
        case (m.CC & N != N) && (m.CC & C == C):
            // positive, carry
            m.CC |= V
        default:
            m.CC &= ^V
    }
    mmu.W8(addr, tmp)
    m.PC += 2
}

func ROL_79(m *M6800, mmu mem.MMU16) {
    addr := mmu.R16(m.PC)
    tmp := mmu.R8(addr)
    wrap := m.CC & C == C
    if tmp & 0x80 == 0x80 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    tmp = tmp << 1
    if wrap {
        tmp |= 0x01
    }
    m.set_NZ8(tmp)
    switch {
        case (m.CC & N == N) && (m.CC & C != C):
            // negative, no carry
            m.CC |= V
        case (m.CC & N != N) && (m.CC & C == C):
            // positive, carry
            m.CC |= V
        default:
            m.CC &= ^V
    }
    mmu.W8(addr, tmp)
    m.PC += 2
}

// *
func DEC_7a(m *M6800, mmu mem.MMU16) {
    addr := mmu.R16(m.PC)
    tmp := mmu.R8(addr)
    if tmp == 0x80 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    tmp -= 1
    m.set_NZ8(tmp)
    mmu.W8(addr, tmp)
    m.PC += 2
}

// *
func INC_7c(m *M6800, mmu mem.MMU16) {
    addr := mmu.R16(m.PC)
    tmp := mmu.R8(addr)
    if tmp == 0x7f {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    tmp += 1
    m.set_NZ8(tmp)
    mmu.W8(addr, tmp)
    m.PC += 2
}

// *
func TST_7d(m *M6800, mmu mem.MMU16) {
    tmp := mmu.R8(mmu.R16(m.PC))
    m.CC &= ^C
    m.CC &= ^V
    m.set_NZ8(tmp)
    m.PC += 2
}

// *
func JMP_7e(m *M6800, mmu mem.MMU16) {
    m.PC = mmu.R16(m.PC)
}

// *
func CLR_7f(m *M6800, mmu mem.MMU16) {
    mmu.W8(mmu.R16(m.PC), 0)
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
    m.PC += 2
}

// *
func SUB_80(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(uint16(m.PC))
    m.A = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

// *
func CMP_81(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(m.PC)
    _ = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

// *
func AND_84(m *M6800, mmu mem.MMU16) {
    m.A &= mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

// *
func BIT_85(m *M6800, mmu mem.MMU16) {
    m.CC &= ^V
    m.set_NZ8(m.A & mmu.R8(m.PC))
    m.PC += 1
}

// *
func LDA_86(m *M6800, mmu mem.MMU16) {
    m.A = mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

// *
func ADC_89(m *M6800, mmu mem.MMU16) {
    augend := m.A
    addend := mmu.R8(m.PC)
    m.A = m.add(augend, addend, true)
    m.PC += 1
}

func ORA_8a(m *M6800, mmu mem.MMU16) {
    m.A |= mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

// *
func ADD_8b(m *M6800, mmu mem.MMU16) {
    augend := m.A
    addend := mmu.R8(m.PC)
    m.A = m.add(augend, addend, false)
    m.PC += 1
}

func CPX_8c(m *M6800, mmu mem.MMU16) {
    tmp := mmu.R16(m.PC)
    // incorrect: affects the carry flag, it should only effect VNZ, but this is easier
    _ = m.sub(uint8(m.X>>8), uint8(tmp>>8), false)
    m.PC += 2
}

// *
func BSR_8d(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    mmu.W16(m.SP-1, m.PC)
    m.SP -= 2
    if offset & 0x80 == 0x80 {
        offset = ^offset + 1
        m.PC -= uint16(offset)
    } else {
        m.PC += uint16(offset)
    }
}

// *
func LDS_8e(m *M6800, mmu mem.MMU16) {
    m.SP = mmu.R16(m.PC)
    m.CC &= ^V
    m.set_NZ16(m.SP)
    m.PC += 2
}

// *
func CMP_91(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(uint16(mmu.R8(m.PC)))
    _ = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

// *
func LDA_96(m *M6800, mmu mem.MMU16) {
    m.A = mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

// *
func STA_97(m *M6800, mmu mem.MMU16) {
    mmu.W8(uint16(mmu.R8(m.PC)), m.A)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

// *
func EOR_98(m *M6800, mmu mem.MMU16) {
    m.A ^= mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

// *
func ORA_9a(m *M6800, mmu mem.MMU16) {
    m.A |= mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

// *
func ADD_9b(m *M6800, mmu mem.MMU16) {
    augend := m.A
    addend := mmu.R8(uint16(mmu.R8(m.PC)))
    sum := m.add(augend, addend, false)
    m.A = sum
    m.PC += 1
}

// * 
func CPX_9c(m *M6800, mmu mem.MMU16) {
    subtrahend := mmu.R16(uint16(mmu.R8(m.PC)))
    difference := m.X - subtrahend

    // overflow
    msign := m.X & 0x8000 == 0x8000
    ssign := subtrahend & 0x8000 == 0x8000
    dsign := difference & 0x8000 == 0x8000
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
    m.set_NZ16(difference)
    m.PC += 1
}

func SUB_a0(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.A = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

// *
func LDA_a6(m *M6800, mmu mem.MMU16) {
    m.A = mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

// *
func STA_a7(m *M6800, mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), m.A)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

// *
func ADD_ab(m *M6800, mmu mem.MMU16) {
    augend := m.A
    addend := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.A = m.add(augend, addend, false)
    m.PC += 1
}

// *
func JSR_ad(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    mmu.W16(m.SP-1, m.PC)
    m.SP -= 2
    m.PC = m.X + uint16(offset)
}

// *
func LDA_b6(m *M6800, mmu mem.MMU16) {
    m.A = mmu.R8(mmu.R16(m.PC))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 2
}

// *
func STA_b7(m *M6800, mmu mem.MMU16) {
    mmu.W8(mmu.R16(m.PC), m.A)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 2
}

// *
func JSR_bd(m *M6800, mmu mem.MMU16) {
    mmu.W16(m.SP-1, m.PC+2)
    m.SP -= 2
    m.PC = mmu.R16(m.PC)
}


func SUB_c0(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(m.PC)
    m.B = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

// *
func CMP_c1(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(m.PC)
    _ = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

func AND_c4(m *M6800, mmu mem.MMU16) {
    m.B &= mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

// *
func LDA_c6(m *M6800, mmu mem.MMU16) {
    m.B = mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

// *
func ADD_cb(m *M6800, mmu mem.MMU16) {
    augend := m.B
    addend := mmu.R8(m.PC)
    m.B = m.add(augend, addend, false)
    m.PC += 1
}

// *
func LDX_ce(m *M6800, mmu mem.MMU16) {
    m.X = mmu.R16(m.PC)
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 2
}

func SUB_d0(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(uint16(mmu.R8(m.PC)))
    m.B = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

// *
func LDA_d6(m *M6800, mmu mem.MMU16) {
    m.B = mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

// *
func STA_d7(m *M6800, mmu mem.MMU16) {
    mmu.W8(uint16(mmu.R8(m.PC)), m.B)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

// *
func LDX_de(m *M6800, mmu mem.MMU16) {
    m.X = mmu.R16(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 1
}

// *
func STX_df(m *M6800, mmu mem.MMU16) {
    mmu.W16(uint16(mmu.R8(m.PC)), m.X)
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 1
}

// *
func LDA_e6(m *M6800, mmu mem.MMU16) {
    m.B = mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

// *
func STA_e7(m *M6800, mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), m.B)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

// *
func LDX_ee(m *M6800, mmu mem.MMU16) {
    m.X = mmu.R16(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 1
}

func STX_ef(m *M6800, mmu mem.MMU16) {
    mmu.W16(m.X + uint16(mmu.R8(m.PC)), m.X)
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 1
}

// *
func LDA_f6(m *M6800, mmu mem.MMU16) {
    m.B = mmu.R8(mmu.R16(m.PC))
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 2
}

// *
func STA_f7(m *M6800, mmu mem.MMU16) {
    mmu.W8(mmu.R16(m.PC), m.B)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 2
}

func LDX_fe(m *M6800, mmu mem.MMU16) {
    m.X = mmu.R16(mmu.R16(m.PC))
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 2
}

func (m*M6800) save_registers(mmu mem.MMU16) {
    // [CC][B ][A ][Xh][Xl][Ph][Pl]  
    mmu.W16(m.SP-1, m.PC)
    mmu.W16(m.SP-3, m.X)
    mmu.W8(m.SP-4, m.A)
    mmu.W8(m.SP-5, m.B)
    mmu.W8(m.SP-6, m.CC)
    m.SP -= 7
}

// flags: --HI NZVC
//        8421 8421

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

// common function to all ADD and ADC opcodes, handles flags, but caller handles advancing PC
func (m *M6800) add(augend, addend uint8, withcarry bool) uint8 {
    sum := augend + addend
    if withcarry && (m.CC & C == C) {
        sum += 1
    }
    augsign := augend & 0x80 == 0x80
    addsign := addend & 0x80 == 0x80
    sumsign := sum & 0x80 == 0x80

    // carry if the following
    // * both operands have high bit (negative)
    // * either of operand has high bit (negative) but the sum doesn't (positive)
    switch {
        case augsign && addsign:
            // negative + negative == _
            m.CC |= C
        case addsign && !sumsign:
            // _ + negative == positive
            m.CC |= C
        case augsign && !sumsign:
            // negative + _ == positive
            m.CC |= C
        default:
            m.CC &= ^C
    }

    // overflow if the following
    // * both operands have high bit (negative), but sum doesn't (positive)
    // * both operands have no high bit (positive), but the sum does (negative)
    switch {
        case !augsign && !addsign && sumsign:
            // positive + positive == negative
            m.CC |= V
        case augsign && addsign && !sumsign:
            // negative + negative == positive
            m.CC |= V
        default:
            m.CC &= ^V
    }

    aughalf := augend & 0x08 == 0x08
    addhalf := addend & 0x08 == 0x08
    sumhalf := sum & 0x08 == 0x08
    // half-carry if the following
    // * both operands have bit 3 set (>7)
    // * either operand has bit 3 clear (<8), but the sum doesn't (>7)  ???XXX???
    switch {
        case aughalf && addhalf:
            m.CC |= H
        case !aughalf && sumhalf:
            m.CC |= H
        case !addhalf && sumhalf:
            m.CC |= H
        default:
            m.CC &= ^H
    }

    m.set_NZ8(sum)
    return sum
}

// common function to all SUB and CMP opcodes, handles flags but caller handles advancing PC
func (m *M6800) sub(minuend, subtrahend uint8, withcarry bool) uint8 {
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
