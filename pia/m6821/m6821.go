package m6821

import (
    "fmt"

    "github.com/bartgrantham/fpemu/ui"
    "github.com/bartgrantham/fpemu/misc/hc55516"
)

/*
TODO:
    IRQn should be derived from Cxn, based on the configuration in CRx
    
*/

const (
    Cx1_0  uint8  = 1 << iota  // MPU interrupt on _Cx1 active transition_ (disable | enable)
    Cx1_1                      // IRQx1 set on _Cx1 edge transition_ (high-to-low | low-to-high)
    DDRx                       // switch for register 0/2 access (DDR | OUTA+INA)
    Cx2_0                      // Cx2 behavior, depends on Cx2_1/Cx2_2
    // Cx2_1 | Cx2_2
    //   0       0     : MPU interrupt on _Cx2 active transition_ (disable | enable)
    //   1       0     : IRQx2 set on _Cx2 edge transition_ (high-to-low | low-to-high)
    //   0       1     : Cx2 returned high on: (next Cx1 transition | E transition on deselect ; high-to-low for A, low-to-high for B)
    //   1       1     : set/reset Cx2 level (low | high)
    Cx2_1
    Cx2_2
    // IRQx2 and IRQx1 are read-only
    IRQx2
    // When Cx2 is an input, IRQx2 goes high on active transition of Cx2, cleared by MPU read of A/B
    // When Cx2 is an output, IRQx2 is always zero (not affected by Cx2 transitions)
    IRQx1                      // IRQx1 goes high on active transition of Cx1, cleared by MPU read of A/B
)

type M6821 struct {
    ORA, ORB    uint8  // output registers
    CRA, CRB    uint8  // control registers
    DDRA, DDRB  uint8  // direction registers (each bit/pin can be set in/out separately)
    IRQA, IRQB  bool   // IRQ state

    INA, INB    uint8  // from the outside world
    CA1, CA2, CB1, CB2 bool
    CVSD        hc55516.CVSD
//    hist []uint8
}

func (m *M6821) String() string {
    fmtstr := "ORA/ORB: %.2X/%.2X ; DDRA/DDRB: %.2X/%.2X ; CRA/CRB: 0x%08b/0x%08b ; IRQA/IRQB: %t/%t ; CA1/CA2/CB1/CB2: %t/%t/%t/%t"
    return fmt.Sprintf(fmtstr, m.ORA, m.ORB, m.DDRA, m.DDRB, m.CRA, m.CRB,
           m.IRQA, m.IRQB, m.CA1, m.CA2, m.CB1, m.CB2)
}

func (m *M6821) R8(addr uint16) uint8 {
    switch addr {
        case 0:
            if m.CRA & DDRx == 0 {
                return m.DDRA
            } else {
                m.CRA &= ^(IRQx2 | IRQx1) // clear interrupt registers
                m.IRQA = false  // FIX
                return (m.ORA & m.DDRA) | (m.INA & ^m.DDRA)  // input + output, appropriately masked
            }
        case 1:
            return m.CRA
        case 2:
            if m.CRB & DDRx == 0 {
                return m.DDRB
            } else {
                m.CRB &= ^(IRQx2 | IRQx1) // clear interrupt registers
                m.IRQB = false  // FIX
ui.Log(fmt.Sprintf("read PIAB: %X %X %X %X %X %X\n", m.ORB, m.INB, m.DDRB, m.ORB & m.DDRB, m.INB & ^m.DDRB, (m.ORB & m.DDRB) | (m.INB & ^m.DDRB)))
                return (m.ORB & m.DDRB) | (m.INB & ^m.DDRB)  // input + output, appropriately masked
            }
        case 3:
            return m.CRB
        default:
            panic(fmt.Sprintf("Unknown register 0x%.4X", addr))
    }
    return 0
}

func (m *M6821) W8(addr uint16, val uint8) {
    switch addr {
        case 0:
            if m.CRA & DDRx == 0 {
                m.DDRA = val
            } else {
                m.ORA = val
            }
        case 1:
            m.CRA = val & 0x3F
            if m.CRA & (Cx2_1 | Cx2_2) == (Cx2_1 | Cx2_2) {
                if m.CRA & Cx2_0 == Cx2_0 {
                    m.CA2 = true
                } else {
                    m.CA2 = false
                }
            }
        case 2:
            if m.CRB & DDRx == 0 {
                m.DDRB = val
            } else {
                m.ORB = val
            }
        case 3:
            m.CRB = val & 0x3F
            if m.CRB & (Cx2_1 | Cx2_2) == (Cx2_1 | Cx2_2) {
                if m.CRB & Cx2_0 == Cx2_0 {
                    if m.CB2 == false {
                        // positive edge
                        m.CVSD.Addbit(m.CA2)
                    }
                    m.CB2 = true
                } else {
                    m.CB2 = false
                }
            }
        default:
            panic(fmt.Sprintf("Unknown register 0x%.4X", addr))
    }
}

func (m *M6821) Read(port uint16) uint8 {
    switch port {
        case 0:
            return m.ORA & m.DDRA
        case 1:
            return m.ORB & m.DDRB
        default:
            panic(fmt.Sprintf("Unknown port 0x%.4X", port))
    }
}

func (m *M6821) Write(port uint16, val uint8) {
    switch port {
        case 0:
            m.INA = val
            m.IRQA = true
            // HUGELY COMPLICATED MESS
        case 1:
            m.INB = val
            m.IRQB = true
            // HUGELY COMPLICATED MESS
        default:
            panic(fmt.Sprintf("Unknown port 0x%.4X", port))
    }
}

func (m *M6821) IRQ(line uint16) bool {
    switch line {
        case 0: return m.IRQA
        case 1: return m.IRQB
        default:
            panic(fmt.Sprintf("Unknown IRQ 0x%.4X", line))
    }
    return false
}

func (m *M6821) Reset() {
}


