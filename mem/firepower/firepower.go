package firepower

import (
    "fmt"
)

type FirepowerMem struct {
    IRAM    [128]uint8
    ORAM    [128]uint8
    IC7     [4096]uint8  // $B000-$BFFF
    IC5     [4096]uint8  // $C000-$CFFF
    IC6     [4096]uint8  // $D000-$DFFF
    IC12    [2048]uint8  // $F800-$FFFF
    // for tracking memory hotspots
    reads   [65536]int
    writes  [65536]int
}


func NewFirepowerMem(ic7, ic5, ic6, ic12 []byte) *FirepowerMem {
    m := FirepowerMem{}
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
        case addr >= 0x400 && addr <= 0x47F:
            val = m.ORAM[addr-0x400]
        case addr >= 0xB000 && addr <= 0xBFFF:
            val = m.IC7[addr-0xB000]
        case addr >= 0xC000 && addr <= 0xCFFF:
            val = m.IC5[addr-0xC000]
        case addr >= 0xD000 && addr <= 0xDFFF:
            val = m.IC6[addr-0xD000]
        case addr >= 0xF800 && addr <= 0xFFFF:
            val = m.IC12[addr-0xF800]
        default:
            panic(fmt.Sprintf("S8 invalid address: $%.4X", addr))
    }
    return val, m.reads[addr], m.writes[addr]
}

func (m *FirepowerMem) R8(addr uint16) uint8 {
    m.reads[addr] += 1
    switch {
        case addr <= 0x7F:
            return m.IRAM[addr]
        case addr >= 0x400 && addr <= 0x47F:
            return m.ORAM[addr-0x400]
        case addr >= 0xB000 && addr <= 0xBFFF:
            return m.IC7[addr-0xB000]
        case addr >= 0xC000 && addr <= 0xCFFF:
            return m.IC5[addr-0xC000]
        case addr >= 0xD000 && addr <= 0xDFFF:
            return m.IC6[addr-0xD000]
        case addr >= 0xF800 && addr <= 0xFFFF:
            return m.IC12[addr-0xF800]
        default:
            panic(fmt.Sprintf("S8 invalid address: $%.4X", addr))
    }
}

func (m *FirepowerMem) W8(addr uint16, val uint8) {
    m.writes[addr] += 1
    switch {
        case addr <= 0x7F:
            m.IRAM[addr] = val
        case addr >= 0x400 && addr <= 0x47F:
            m.ORAM[addr-0x400] = val
        case addr >= 0xB000 && addr <= 0xBFFF:
            m.IC7[addr-0xB000] = val
        case addr >= 0xC000 && addr <= 0xCFFF:
            m.IC5[addr-0xC000] = val
        case addr >= 0xD000 && addr <= 0xDFFF:
            m.IC6[addr-0xD000] = val
        case addr >= 0xF800 && addr <= 0xFFFF:
            m.IC12[addr-0xF800] = val
        default:
            panic(fmt.Sprintf("S8 invalid address: $%.4X", addr))
    }
    return
}

func (m *FirepowerMem) R16(addr uint16) uint16 {
    high := uint16(m.R8(addr))
    low  := uint16(m.R8(addr+1))
    return (high<<8) + low
}

