package m6800

import (
    "fmt"

    "github.com/bartgrantham/fpemu/mem"
)

// Motorola_M6800_Programming_Reference_Manual_M68PRM(D)_Nov76.pdf has a LOT of detail on flags

/*
* overflow of A+B=R is defined as: V = (A:7 == B:7) & (R:7 != A:7)  (A and B same sign, R opposite)
* overflow of A-B=R is defined as: V = (A:7 == !B:7) & (R:7 != A:7) (A and -B same sign, R opposite)

* REI restores the previous interrupt flag from the stack by restoring CC
* note that calling CLI at the beginning of an interrupt allows a higher priority interrupt to occur, because the registers will be saved if another interrupt happens.  The manual explicitly mentions this nesting.

lasercue 0x09 (lines 2724-4374):

8

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
    INVALD, NOP_01, INVALD, INVALD, INVALD, INVALD, TAP_06, TPA_07, INX_08, DEX_09, CLV_0A, SEV_0B, CLC_0C, SEC_0D, CLI_0E, SEI_0F, //00
    SBA_10, CMP_11, INVALD, INVALD, INVALD, INVALD, TAB_16, TBA_17, INVALD, UNIMPL, INVALD, ABA_1B, INVALD, INVALD, INVALD, INVALD, //10
    BRA_20, INVALD, BHI_22, BLS_23, BCC_24, BCS_25, BNE_26, BEQ_27, BVC_28, UNIMPL, BPL_2A, BMI_2B, UNIMPL, BLT_2D, BGT_2E, UNIMPL, //20
    UNIMPL, UNIMPL, PUL_32, PUL_33, UNIMPL, UNIMPL, PSH_36, PSH_37, INVALD, RTS_39, INVALD, UNIMPL, INVALD, INVALD, UNIMPL, UNIMPL, //30
    UNIMPL, INVALD, INVALD, COM_43, LSR_44, INVALD, UNIMPL, ASR_47, ASL_48, ROL_49, DEC_4A, INVALD, INC_4C, TST_4D, INVALD, CLR_4F, //40
    NEG_50, INVALD, INVALD, COM_53, LSR_54, INVALD, ROR_56, UNIMPL, ASL_58, ROL_59, DEC_5A, INVALD, INC_5C, TST_5D, INVALD, CLR_5F, //50
    NEG_60, INVALD, INVALD, UNIMPL, UNIMPL, INVALD, UNIMPL, UNIMPL, UNIMPL, UNIMPL, DEC_6A, INVALD, UNIMPL, TST_6D, JMP_6E, CLR_6F, //60
    UNIMPL, INVALD, INVALD, COM_73, LSR_74, INVALD, ROR_76, UNIMPL, ASL_78, ROL_79, DEC_7A, INVALD, INC_7C, TST_7D, JMP_7E, CLR_7F, //70
    SUB_80, CMP_81, SBC_82, INVALD, AND_84, BIT_85, LDA_86, INVALD, UNIMPL, ADC_89, ORA_8A, ADD_8B, CPX_8C, BSR_8D, LDS_8E, INVALD, //80
    SUB_90, CMP_91, SBC_92, INVALD, UNIMPL, UNIMPL, LDA_96, STA_97, EOR_98, ADC_99, ORA_9A, ADD_9B, CPX_9C, INVALD, LDX_9E, STX_9F, //90
    SUB_A0, UNIMPL, UNIMPL, INVALD, UNIMPL, UNIMPL, LDA_A6, STA_A7, UNIMPL, UNIMPL, UNIMPL, ADD_AB, UNIMPL, JSR_AD, UNIMPL, UNIMPL, //A0
    UNIMPL, CMP_B1, UNIMPL, INVALD, UNIMPL, UNIMPL, LDA_B6, STA_B7, UNIMPL, UNIMPL, UNIMPL, UNIMPL, UNIMPL, JSR_BD, UNIMPL, UNIMPL, //B0
    SUB_C0, CMP_C1, SBC_C2, INVALD, AND_C4, UNIMPL, LDA_C6, INVALD, UNIMPL, ADC_C9, UNIMPL, ADD_CB, INVALD, INVALD, LDX_CE, INVALD, //C0
    SUB_D0, CMP_D1, SBC_D2, INVALD, AND_D4, UNIMPL, LDA_D6, STA_D7, EOR_D8, ADC_D9, UNIMPL, ADD_DB, INVALD, INVALD, LDX_DE, STX_DF, //D0
    UNIMPL, CMP_E1, UNIMPL, INVALD, AND_E4, BIT_E5, LDA_E6, STA_E7, UNIMPL, UNIMPL, UNIMPL, ADD_EB, INVALD, INVALD, LDX_EE, STX_EF, //E0
    UNIMPL, CMP_F1, UNIMPL, INVALD, UNIMPL, UNIMPL, LDA_F6, STA_F7, UNIMPL, UNIMPL, UNIMPL, ADD_FB, INVALD, INVALD, LDX_FE, STX_FF, //F0
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

// No Operation
func NOP_01(m *M6800, mmu mem.MMU16) {
}

// Transfer A to Status Register
func TAP_06(m *M6800, mmu mem.MMU16) {
    m.CC = m.A & 0x3f
    // Condition Codes not affected (pg. A-72)
}

// Transfer Status Register to A
func TPA_07(m *M6800, mmu mem.MMU16) {
    m.A = m.CC | 0xC0
    // Condition Codes not affected (pg. A-70)
}

// Increment Index Register X 
func INX_08(m *M6800, mmu mem.MMU16) {
    m.X += 1
    if m.X == 0 {
        m.CC |= Z
    } else {
        m.CC &= Z
    }
}

// Decrement Index Register X
func DEX_09(m *M6800, mmu mem.MMU16) {
    m.X -= 1
    if m.X == 0 {
        m.CC |= Z
    } else {
        m.CC &= Z
    }
}

// Clear Overflow flag
func CLV_0A(m *M6800, mmu mem.MMU16) {
    m.CC &= ^V
    // Only overflow affected (pg A-30)
}

// Set Overflow Flag
func SEV_0B(m *M6800, mmu mem.MMU16) {
    m.CC |= V
    // Only overflow affected (pg A-62)
}

// Clear Carry Flag
func CLC_0C(m *M6800, mmu mem.MMU16) {
    m.CC &= ^C
    // Only carry affected (pg A-27)
}

// Set Carry Flag
func SEC_0D(m *M6800, mmu mem.MMU16) {
    m.CC |= C
    // Only carry affected (pg A-60)
}

// Clear Interrupt Enable
func CLI_0E(m *M6800, mmu mem.MMU16) {
    m.CC &= ^I
    // Only interrupt enable affected (pg A-28)
}

// Set Interrupt Enable
func SEI_0F(m *M6800, mmu mem.MMU16) {
    m.CC |= I
    // Only interrupt enable affected (pg A-61)
}

// Subtract B from A
func SBA_10(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := m.B
    m.A = m.sub(minuend, subtrahend, false)
}

// Compare B from A
func CMP_11(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := m.B
    _ = m.sub(minuend, subtrahend, false)
}

// Transfer A to B
func TAB_16(m *M6800, mmu mem.MMU16) {
    m.B = m.A
    m.CC &= ^V
    m.set_NZ8(m.B)
}

// Transfer B to A
func TBA_17(m *M6800, mmu mem.MMU16) {
    m.A = m.B
    m.CC &= ^V
    m.set_NZ8(m.A)
}

// Add B to A
func ABA_1B(m *M6800, mmu mem.MMU16) {
    augend := m.A
    addend := m.B
    sum := m.add(augend, addend, false)
    m.A = sum
}

// Unconditional Branch
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

// Branch if "higher" (carry and zero are clear)
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

// Branch if "less than or same" (carry or zero are set)
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

// Branch if carry clear
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

// Branch if carry set
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

// Branch if not equal
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

// Branch if equal
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

// Branch if overflow clear
func BVC_28(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    if m.CC & V == 0 {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// untested - Branch if overflow set
func BVS_29(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    if m.CC & V == V {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// Branch if plus (positive)
func BPL_2A(m *M6800, mmu mem.MMU16) {
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

// Branch if minus (negative)
func BMI_2B(m *M6800, mmu mem.MMU16) {
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

// Branch if "less than" (either overflow or negative set)
func BLT_2D(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    v := m.CC & V == V
    n := m.CC & N == N
    if ( v && !n ) || ( n && !v) {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}

// Branch if "greater than" (not zero, and either negative and overflow, or neither negative and overflow)
func BGT_2E(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    z := m.CC & Z == Z
    v := m.CC & V == V
    n := m.CC & N == N
    if !z && ((n && v) || (!n && !v)) {
        if offset & 0x80 == 0x80 {
            offset = ^offset + 1
            m.PC -= uint16(offset)
        } else {
            m.PC += uint16(offset)
        }
    }
}


// Pull from stack to A
func PUL_32(m *M6800, mmu mem.MMU16) {
    m.SP += 1
    m.A = mmu.R8(m.SP)
}

// Pull from stack to B
func PUL_33(m *M6800, mmu mem.MMU16) {
    m.SP += 1
    m.B = mmu.R8(m.SP)
}

// Push from A to stack
func PSH_36(m *M6800, mmu mem.MMU16) {
    mmu.W8(m.SP, m.A)
    m.SP -= 1
}

// Push from B to stack
func PSH_37(m *M6800, mmu mem.MMU16) {
    mmu.W8(m.SP, m.B)
    m.SP -= 1
}

func RTS_39(m *M6800, mmu mem.MMU16) {
    m.PC = mmu.R16(m.SP+1)
    m.SP += 2
}

// untested - Negate A
func NEG_40(m *M6800, mmu mem.MMU16) {
    m.A = ^m.A + 1
    m.set_NZ8(m.A)
    if m.A == 0x80 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    if m.A != 0x00 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
}

// Compliment A
func COM_43(m *M6800, mmu mem.MMU16) {
    m.A = ^m.A //0xff - m.A
    m.CC |= C
    m.CC &= ^V
    m.set_NZ8(m.A)
}

// Logical Shift Right
func LSR_44(m *M6800, mmu mem.MMU16) {
    if m.A & 0x01 == 0x01 {
        m.CC |= C
        m.CC |= V
    }
    m.A >>= 1
    m.set_NZ8(m.A)
}

// untested - Rotate Right Through Carry
func ROR_46(m *M6800, mmu mem.MMU16) {
    wrap := m.CC & C == C
    if m.A & 0x01 == 0x01 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    m.A = m.A >> 1
    if wrap {
        m.A |= 0x80
    }
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

// Arithmetic Shift Right
func ASR_47(m *M6800, mmu mem.MMU16) {
    if m.A & 0x01 == 0x81 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    m.A = m.A >> 1
    m.A |= (m.A & 0x40) << 1    // "Bit 7 is held constant." (pg. A-8)
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


// Arithmetic Shift Left
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

// Rotate Left Through Carry
func ROL_49(m *M6800, mmu mem.MMU16) {
    wrap := m.CC & C == C
    if m.A & 0x80 == 0x80 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    m.A = m.A << 1
    if wrap {
        m.A |= 0x01
    }
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


// Decrement A
func DEC_4A(m *M6800, mmu mem.MMU16) {
    if m.A == 0x80 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.A -= 1
    m.set_NZ8(m.A)
}

// Increment A
func INC_4C(m *M6800, mmu mem.MMU16) {
    if m.A == 0x7f {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.A += 1
    m.set_NZ8(m.A)
}

// Test A
func TST_4D(m *M6800, mmu mem.MMU16) {
    m.CC &= ^C
    m.CC &= ^V
    m.set_NZ8(m.A)
}

// Clear A
func CLR_4F(m *M6800, mmu mem.MMU16) {
    m.A = 0
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
}

// untested - Negate B
func NEG_50(m *M6800, mmu mem.MMU16) {
    m.B = ^m.B + 1
    m.set_NZ8(m.B)
    if m.B == 0x80 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    if m.B != 0x00 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
}

// Compliment B
func COM_53(m *M6800, mmu mem.MMU16) {
    m.B = 0xff - m.B
    m.CC |= C
    m.CC &= ^V
    m.set_NZ8(m.B)
}

// Logical Shift Right
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

// Rotate Right Through Carry
func ROR_56(m *M6800, mmu mem.MMU16) {
    wrap := m.CC & C == C
    if m.B & 0x01 == 0x01 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    m.B = m.B >> 1
    if wrap {
        m.B |= 0x80
    }
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

// Arithmetic Shift Left
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

// Rotate Left Through Carry
func ROL_59(m *M6800, mmu mem.MMU16) {
    wrap := m.CC & C == C
    if m.B & 0x80 == 0x80 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    m.B = m.B << 1
    if wrap {
        m.B |= 0x01
    }
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


// Decrement B
func DEC_5A(m *M6800, mmu mem.MMU16) {
    if m.B == 0x80 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.B -= 1
    m.set_NZ8(m.B)
}


// Increment B
func INC_5C(m *M6800, mmu mem.MMU16) {
    if m.B == 0x7f {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    m.B += 1
    m.set_NZ8(m.B)
}


// Test B
func TST_5D(m *M6800, mmu mem.MMU16) {
    m.CC &= ^C
    m.CC &= ^V
    m.set_NZ8(m.B)
}


// Clear B
func CLR_5F(m *M6800, mmu mem.MMU16) {
    m.B = 0
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
}


// Negate Memory Location
func NEG_60(m *M6800, mmu mem.MMU16) {
    tmp := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    tmp = ^tmp + 1
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), tmp)
    m.set_NZ8(tmp)
    if tmp == 0x80 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    if tmp != 0x00 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    m.PC += 1
}

// Decrement Memory Location
func DEC_6A(m *M6800, mmu mem.MMU16) {
    tmp := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    if tmp == 0x80 {
        m.CC |= V
    } else {
        m.CC &= ^V
    }
    tmp -= 1
    m.set_NZ8(tmp)
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), tmp)
    m.PC += 1
}

func TST_6D(m *M6800, mmu mem.MMU16) {
    tmp := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^C
    m.CC &= ^V
    m.set_NZ8(tmp)
    m.PC += 1
}

func JMP_6E(m *M6800, mmu mem.MMU16) {
    m.PC = m.X + uint16(mmu.R8(m.PC))
}

func CLR_6F(m *M6800, mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), 0)
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
    m.PC += 1
}

func COM_73(m *M6800, mmu mem.MMU16) {
    addr := mmu.R16(m.PC)
    tmp := ^mmu.R8(addr)
    m.CC |= C
    m.CC &= ^V
    m.set_NZ8(tmp)
    mmu.W8(addr, tmp)
    m.PC += 2
}

func LSR_74(m *M6800, mmu mem.MMU16) {
    addr := mmu.R16(m.PC)
    tmp := mmu.R8(addr)
    if tmp & 0x01 == 0x01 {
        m.CC |= C
        m.CC |= V
    }
    tmp >>= 1
    m.set_NZ8(tmp)
    mmu.W8(addr, tmp)
    m.PC += 2
}


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

// Arithmetic Shift Left
func ASL_78(m *M6800, mmu mem.MMU16) {
    addr := mmu.R16(m.PC)
    tmp := mmu.R8(addr)
    if tmp & 0x80 == 0x80 {
        m.CC |= C
    } else {
        m.CC &= ^C
    }
    tmp = tmp << 1
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

func DEC_7A(m *M6800, mmu mem.MMU16) {
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

func INC_7C(m *M6800, mmu mem.MMU16) {
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

func TST_7D(m *M6800, mmu mem.MMU16) {
    tmp := mmu.R8(mmu.R16(m.PC))
    m.CC &= ^C
    m.CC &= ^V
    m.set_NZ8(tmp)
    m.PC += 2
}

func JMP_7E(m *M6800, mmu mem.MMU16) {
    m.PC = mmu.R16(m.PC)
}

func CLR_7F(m *M6800, mmu mem.MMU16) {
    mmu.W8(mmu.R16(m.PC), 0)
    // set Z, clear NVC
    m.CC |= Z
    m.CC &= ^N
    m.CC &= ^V
    m.CC &= ^C
    m.PC += 2
}

func SUB_80(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(uint16(m.PC))
    m.A = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

func CMP_81(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(m.PC)
    _ = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

func SBC_82(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(uint16(m.PC))
    m.A = m.sub(minuend, subtrahend, true)
    m.PC += 1
}

func AND_84(m *M6800, mmu mem.MMU16) {
    m.A &= mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func BIT_85(m *M6800, mmu mem.MMU16) {
    m.CC &= ^V
    m.set_NZ8(m.A & mmu.R8(m.PC))
    m.PC += 1
}

func LDA_86(m *M6800, mmu mem.MMU16) {
    m.A = mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func ADC_89(m *M6800, mmu mem.MMU16) {
    augend := m.A
    addend := mmu.R8(m.PC)
    m.A = m.add(augend, addend, true)
    m.PC += 1
}

func ORA_8A(m *M6800, mmu mem.MMU16) {
    m.A |= mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func ADD_8B(m *M6800, mmu mem.MMU16) {
    augend := m.A
    addend := mmu.R8(m.PC)
    m.A = m.add(augend, addend, false)
    m.PC += 1
}

func CPX_8C(m *M6800, mmu mem.MMU16) {
    tmp := mmu.R16(m.PC)
    // incorrect: affects the carry flag, it should only effect VNZ, but this is easier
    _ = m.sub(uint8(m.X>>8), uint8(tmp>>8), false)
    m.PC += 2
}

func BSR_8D(m *M6800, mmu mem.MMU16) {
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

// Load Stack Pointer
func LDS_8E(m *M6800, mmu mem.MMU16) {
    m.SP = mmu.R16(m.PC)
    m.CC &= ^V
    m.set_NZ16(m.SP)
    m.PC += 2
}

func SUB_90(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(uint16(mmu.R8(m.PC)))
    m.A = m.sub(minuend, subtrahend, false)
    m.PC += 1
}


func CMP_91(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(uint16(mmu.R8(m.PC)))
    _ = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

func SBC_92(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(uint16(mmu.R8(m.PC)))
    m.A = m.sub(minuend, subtrahend, true)
    m.PC += 1
}

func LDA_96(m *M6800, mmu mem.MMU16) {
    m.A = mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func STA_97(m *M6800, mmu mem.MMU16) {
    mmu.W8(uint16(mmu.R8(m.PC)), m.A)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func EOR_98(m *M6800, mmu mem.MMU16) {
    m.A ^= mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func ADC_99(m *M6800, mmu mem.MMU16) {
    augend := mmu.R8(uint16(mmu.R8(m.PC)))
    addend := mmu.R8(m.PC)
    m.A = m.add(augend, addend, true)
    m.PC += 1
}

func ORA_9A(m *M6800, mmu mem.MMU16) {
    m.A |= mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func ADD_9B(m *M6800, mmu mem.MMU16) {
    augend := m.A
    addend := mmu.R8(uint16(mmu.R8(m.PC)))
    sum := m.add(augend, addend, false)
    m.A = sum
    m.PC += 1
}

func CPX_9C(m *M6800, mmu mem.MMU16) {
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

// Load Stack Pointer
func LDX_9E(m *M6800, mmu mem.MMU16) {
    m.SP = mmu.R16(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ16(m.SP)
    m.PC += 1
}

// Store Stack Pointer
func STX_9F(m *M6800, mmu mem.MMU16) {
    mmu.W16(uint16(mmu.R8(m.PC)), m.SP)
    m.CC &= ^V
    m.set_NZ16(m.SP)
    m.PC += 1
}

func SUB_A0(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.A = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

func LDA_A6(m *M6800, mmu mem.MMU16) {
    m.A = mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func STA_A7(m *M6800, mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), m.A)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 1
}

func ADD_AB(m *M6800, mmu mem.MMU16) {
    augend := m.A
    addend := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.A = m.add(augend, addend, false)
    m.PC += 1
}

func JSR_AD(m *M6800, mmu mem.MMU16) {
    offset := mmu.R8(m.PC)
    m.PC += 1
    mmu.W16(m.SP-1, m.PC)
    m.SP -= 2
    m.PC = m.X + uint16(offset)
}

// untested
func CMP_B1(m *M6800, mmu mem.MMU16) {
    minuend := m.A
    subtrahend := mmu.R8(mmu.R16(m.PC))
    _ = m.sub(minuend, subtrahend, false)
    m.PC += 2
}

func LDA_B6(m *M6800, mmu mem.MMU16) {
    m.A = mmu.R8(mmu.R16(m.PC))
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 2
}

func STA_B7(m *M6800, mmu mem.MMU16) {
    mmu.W8(mmu.R16(m.PC), m.A)
    m.CC &= ^V
    m.set_NZ8(m.A)
    m.PC += 2
}

func JSR_BD(m *M6800, mmu mem.MMU16) {
    mmu.W16(m.SP-1, m.PC+2)
    m.SP -= 2
    m.PC = mmu.R16(m.PC)
}


func SUB_C0(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(m.PC)
    m.B = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

func CMP_C1(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(m.PC)
    _ = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

func SBC_C2(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(uint16(m.PC))
    m.B = m.sub(minuend, subtrahend, true)
    m.PC += 1
}

func AND_C4(m *M6800, mmu mem.MMU16) {
    m.B &= mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func LDA_C6(m *M6800, mmu mem.MMU16) {
    m.B = mmu.R8(m.PC)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func ADC_C9(m *M6800, mmu mem.MMU16) {
    augend := m.B
    addend := mmu.R8(m.PC)
    m.B = m.add(augend, addend, true)
    m.PC += 1
}


func ADD_CB(m *M6800, mmu mem.MMU16) {
    augend := m.B
    addend := mmu.R8(m.PC)
    m.B = m.add(augend, addend, false)
    m.PC += 1
}

func LDX_CE(m *M6800, mmu mem.MMU16) {
    m.X = mmu.R16(m.PC)
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 2
}

func SUB_D0(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(uint16(mmu.R8(m.PC)))
    m.B = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

func CMP_D1(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(uint16(mmu.R8(m.PC)))
    _ = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

// untested
func SBC_D2(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(uint16(mmu.R8(m.PC)))
    m.B = m.sub(minuend, subtrahend, true)
    m.PC += 1
}

func AND_D4(m *M6800, mmu mem.MMU16) {
    m.B &= mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func LDA_D6(m *M6800, mmu mem.MMU16) {
    m.B = mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func STA_D7(m *M6800, mmu mem.MMU16) {
    mmu.W8(uint16(mmu.R8(m.PC)), m.B)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func EOR_D8(m *M6800, mmu mem.MMU16) {
    m.B ^= mmu.R8(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func ADC_D9(m *M6800, mmu mem.MMU16) {
    augend := mmu.R8(uint16(mmu.R8(m.PC)))
    addend := mmu.R8(m.PC)
    m.B = m.add(augend, addend, true)
    m.PC += 1
}

func ADD_DB(m *M6800, mmu mem.MMU16) {
    augend := m.B
    addend := mmu.R8(uint16(mmu.R8(m.PC)))
    sum := m.add(augend, addend, false)
    m.B = sum
    m.PC += 1
}


// Load X
func LDX_DE(m *M6800, mmu mem.MMU16) {
    m.X = mmu.R16(uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 1
}

// Store X
func STX_DF(m *M6800, mmu mem.MMU16) {
    mmu.W16(uint16(mmu.R8(m.PC)), m.X)
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 1
}

func CMP_E1(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    _ = m.sub(minuend, subtrahend, false)
    m.PC += 1
}

func AND_E4(m *M6800, mmu mem.MMU16) {
    m.B &= mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func BIT_E5(m *M6800, mmu mem.MMU16) {
    m.CC &= ^V
    m.set_NZ8(m.B & mmu.R8(m.X + uint16(mmu.R8(m.PC))))
    m.PC += 1
}

func LDA_E6(m *M6800, mmu mem.MMU16) {
    m.B = mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func STA_E7(m *M6800, mmu mem.MMU16) {
    mmu.W8(m.X + uint16(mmu.R8(m.PC)), m.B)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 1
}

func ADD_EB(m *M6800, mmu mem.MMU16) {
    augend := m.B
    addend := mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.B = m.add(augend, addend, false)
    m.PC += 1
}


func LDX_EE(m *M6800, mmu mem.MMU16) {
    m.X = mmu.R16(m.X + uint16(mmu.R8(m.PC)))
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 1
}

func STX_EF(m *M6800, mmu mem.MMU16) {
    mmu.W16(m.X + uint16(mmu.R8(m.PC)), m.X)
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 1
}

func CMP_F1(m *M6800, mmu mem.MMU16) {
    minuend := m.B
    subtrahend := mmu.R8(mmu.R16(m.PC))
    _ = m.sub(minuend, subtrahend, false)
    m.PC += 2
}

func LDA_F6(m *M6800, mmu mem.MMU16) {
    m.B = mmu.R8(mmu.R16(m.PC))
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 2
}

func STA_F7(m *M6800, mmu mem.MMU16) {
    mmu.W8(mmu.R16(m.PC), m.B)
    m.CC &= ^V
    m.set_NZ8(m.B)
    m.PC += 2
}

func ADD_FB(m *M6800, mmu mem.MMU16) {
    augend := m.B
    addend := mmu.R8(mmu.R16(m.PC)) //mmu.R8(m.X + uint16(mmu.R8(m.PC)))
    m.B = m.add(augend, addend, false)
    m.PC += 1
}


func LDX_FE(m *M6800, mmu mem.MMU16) {
    m.X = mmu.R16(mmu.R16(m.PC))
    m.CC &= ^V
    m.set_NZ16(m.X)
    m.PC += 2
}

func STX_FF(m *M6800, mmu mem.MMU16) {
    mmu.W16(mmu.R16(m.PC), m.X)
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
    if withcarry && (m.CC & C == C) {
        subtrahend += 1
    }

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
