package firepower

import (
    "fmt"
//"log"
    "github.com/bartgrantham/fpemu/pia"
)

type FirepowerMem struct {
    IRAM    [128]uint8   // $0000-$007F
//    ORAM    [128]uint8   // $0400-$047F
    IC7     [4096]uint8  // $B000-$BFFF
    IC5     [4096]uint8  // $C000-$CFFF
    IC6     [4096]uint8  // $D000-$DFFF
    IC12    [2048]uint8  // $F800-$FFFF
    PIA     pia.PIA
    // for tracking memory hotspots
    reads   [65536]int
    writes  [65536]int
}


func NewFirepowerMem(ic7, ic5, ic6, ic12 []byte, pia pia.PIA) *FirepowerMem {
    m := FirepowerMem{PIA:pia}
    copy(m.IC7[:], ic7)
    copy(m.IC5[:], ic5)
    copy(m.IC6[:], ic6)
    copy(m.IC12[:], ic12)
    return &m
}

func (m *FirepowerMem) ClearPeekCounts() {
    for i, _ := range m.reads {
        m.reads[i] = 0
        m.writes[i] = 0
    }
}

func (m *FirepowerMem) Peek8(addr uint16) (uint8, int, int) {
    val := uint8(0)
    switch {
        case addr <= 0x7F:
            val = m.IRAM[addr]
//        case addr >= 0x400 && addr <= 0x47F:
//            val = m.ORAM[addr-0x400]
        case addr >= 0x400 && addr <= 0x403:
            m.PIA.R8(addr-0x400)
        case addr >= 0xB000 && addr <= 0xBFFF:
            val = m.IC7[addr-0xB000]
        case addr >= 0xC000 && addr <= 0xCFFF:
            val = m.IC5[addr-0xC000]
        case addr >= 0xD000 && addr <= 0xDFFF:
            val = m.IC6[addr-0xD000]
        case addr >= 0xF800 && addr <= 0xFFFF:
            val = m.IC12[addr-0xF800]
        default:
            panic(fmt.Sprintf("Peek8 invalid address: $%.4X", addr))
    }
    return val, m.reads[addr], m.writes[addr]
}

func (m *FirepowerMem) R8(addr uint16) uint8 {
    m.reads[addr] += 1
    switch {
        case addr <= 0x7F:
            return m.IRAM[addr]
//        case addr >= 0x400 && addr <= 0x47F:
//            return m.ORAM[addr-0x400]
        case addr >= 0x400 && addr <= 0x403:
            return m.PIA.R8(addr-0x400)
        case addr >= 0xB000 && addr <= 0xBFFF:
            return m.IC7[addr-0xB000]
        case addr >= 0xC000 && addr <= 0xCFFF:
            return m.IC5[addr-0xC000]
        case addr >= 0xD000 && addr <= 0xDFFF:
            return m.IC6[addr-0xD000]
        case addr >= 0xF800 && addr <= 0xFFFF:
            return m.IC12[addr-0xF800]
        default:
            panic(fmt.Sprintf("R8 invalid address: $%.4X", addr))
    }
}

func (m *FirepowerMem) W8(addr uint16, val uint8) {
    m.writes[addr] += 1
    switch {
        case addr <= 0x7F:
            m.IRAM[addr] = val
//        case addr >= 0x400 && addr <= 0x47F:
//            m.ORAM[addr-0x400] = val
        case addr >= 0x400 && addr <= 0x403:
            m.PIA.W8(addr-0x400, val)
//        case addr >= 0xB000 && addr <= 0xBFFF:
//            m.IC7[addr-0xB000] = val
//        case addr >= 0xC000 && addr <= 0xCFFF:
//            m.IC5[addr-0xC000] = val
//        case addr >= 0xD000 && addr <= 0xDFFF:
//            m.IC6[addr-0xD000] = val
//        case addr >= 0xF800 && addr <= 0xFFFF:
//            m.IC12[addr-0xF800] = val
        default:
            panic(fmt.Sprintf("W8 invalid address: $%.4X", addr))
    }
    return
}

func (m *FirepowerMem) R16(addr uint16) uint16 {
    var high, low uint8
    m.reads[addr] += 1
    m.reads[addr+1] += 1
    switch {
        case addr <= 0x7E:
            high = m.IRAM[addr]
            low  = m.IRAM[addr+1]
//        case addr >= 0x400 && addr <= 0x47E:
//            high = m.ORAM[addr-0x400]
//            low  = m.ORAM[addr-0x400+1]
        case addr >= 0xB000 && addr <= 0xBFFE:
//            panic("B000")
            high = m.IC7[addr-0xB000]
            low  = m.IC7[addr-0xB000+1]
        case addr >= 0xC000 && addr <= 0xCFFE:
//            panic("C000")
            high = m.IC5[addr-0xC000]
            low  = m.IC5[addr-0xC000+1]
        case addr >= 0xD000 && addr <= 0xDFFE:
//            panic("D000")
            high = m.IC6[addr-0xD000]
            low  = m.IC6[addr-0xD000+1]
        case addr >= 0xF800 && addr <= 0xFFFE:
            high = m.IC12[addr-0xF800]
            low  = m.IC12[addr-0xF800+1]
        default:
            panic(fmt.Sprintf("R16 invalid address: $%.4X", addr))
    }
    return (uint16(high)<<8) + uint16(low)
}

func (m *FirepowerMem) W16(addr uint16, val uint16) {
    m.writes[addr] += 1
    m.writes[addr+1] += 1
    switch {
        case addr <= 0x7E:
            m.IRAM[addr] = uint8(val>>8)
            m.IRAM[addr+1] = uint8(val)
//        case addr >= 0x400 && addr <= 0x47E:
//            m.ORAM[addr-0x400] = uint8(val>>8)
//            m.ORAM[addr-0x400+1] = uint8(val)
        default:
            // everything else is ROM
            panic(fmt.Sprintf("W16 invalid address: $%.4X", addr))
    }
}

