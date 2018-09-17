package firepower

import (
    "fmt"
)

type FirepowerMem struct {
    IRAM  [128]uint8
    ORAM  [128]uint8
    ROM   [2048]uint8
}

func NewFirepowerMem(rom []byte) *FirepowerMem {
    if len(rom) != 2048 {
        return nil
    }
    fpm := FirepowerMem{}
    for i, b := range rom {
        fpm.ROM[i] = uint8(b)
    }
    return &fpm
}

func Dump(ram [128]uint8) string {
    output := ""
    for i, b := range ram {
        output += fmt.Sprintf("%.2X", b)
        switch {
            case (i+1) % 32 == 0:  output += "\n"
            case (i+1) % 8  == 0:  output += "  "
            default:  output += " "
        }
    }
    return output
}

func (m *FirepowerMem) R8(addr uint16) uint8 {
    addr &= 0xFFF
    switch {
        case addr < 0x7F:
            addr %= uint16(len(m.IRAM))
            return m.IRAM[addr]
        case addr >= 0x400 && addr < 0x47F:
            addr %= uint16(len(m.ORAM))
            return m.ORAM[addr]
        default: //addr >= 0x800 && addr < 0x1000:
            addr %= uint16(len(m.ROM))
            return m.ROM[addr]
    }
}

func (m *FirepowerMem) W8(addr uint16, val uint8) {
    addr &= 0xFFF
    switch {
        case addr < 0x7F:
            addr %= uint16(len(m.IRAM))
            m.IRAM[addr] = val
        case addr >= 0x400 && addr < 0x47F:
            addr %= uint16(len(m.ORAM))
            m.ORAM[addr] = val
        default: //addr >= 0x800 && addr < 0x1000:
            addr %= uint16(len(m.ROM))
            m.ROM[addr] = val
    }
    return
}

func (m *FirepowerMem) R16(addr uint16) uint16 {
    addr &= 0xFFF
    switch {
        case addr < 0x7F:
            modmask := uint16(len(m.IRAM))
            addr %= modmask
            addr1 := (addr+1) % modmask
            return (uint16(m.IRAM[addr])<<8) + uint16(m.IRAM[addr1])
        case addr >= 0x400 && addr < 0x47F:
            modmask := uint16(len(m.ORAM))
            addr %= modmask
            addr1 := (addr+1) % modmask
            return (uint16(m.ORAM[addr])<<8) + uint16(m.ORAM[addr1])
        default: //addr >= 0x800 && addr < 0x1000:
            modmask := uint16(len(m.ROM))
            addr %= modmask
            addr1 := (addr+1) % modmask
            return (uint16(m.ROM[addr])<<8) + uint16(m.ROM[addr1])
    }
    return 0
}

