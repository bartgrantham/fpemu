package d8224

import (
    "fmt"
    "log"
    "github.com/bartgrantham/fpemu/pia"
)

type D8224Mem struct {
    PIA     pia.PIA
    RxM     [1<<16]uint8  // $0000-$FFFF
    valid   [1<<16]bool
    reads   [1<<16]int
    writes  [1<<16]int
}

func NewD8224Mem(pia pia.PIA) *D8224Mem {
    d := D8224Mem{PIA:pia}
    // the 6802 has 128b built-in
    for i:=0; i<128; i++ {
        d.valid[i] = true
    }
    return &d
}

func (d *D8224Mem) Mount(addr uint16, data []byte) error {
    if int(addr) + len(data) > 1<<16 {
        return fmt.Errorf("invalid mount")
    }
    if int(addr) < 128 {
        return fmt.Errorf("invalid mount")
    }
    for i, b := range data {
        d.RxM[int(addr) + i] = b
        d.valid[int(addr) + i] = true
    }
    return nil
}

func (d *D8224Mem) ClearPeekCounts() {
    for i, _ := range d.reads {
        d.reads[i] = 0
        d.writes[i] = 0
    }
}

func (d *D8224Mem) Peek8(addr uint16) (uint8, int, int) {
    val := uint8(0)
    switch {
        case addr >= 0x400 && addr <= 0x403:
            val = d.PIA.R8(addr-0x400)
        case d.valid[addr]:
            val = d.RxM[addr]
        default:
            log.Panicf("Peek8 invalid address: $%.4X\n", addr)
    }
    return val, d.reads[addr], d.writes[addr]
}

func (d *D8224Mem) R8(addr uint16) uint8 {
    d.reads[addr] += 1
    val := uint8(0)
    switch {
        case addr >= 0x400 && addr <= 0x403:
            val = d.PIA.R8(addr-0x400)
        case d.valid[addr]:
            val = d.RxM[addr]
        case addr == uint16(0xeffd) || addr == uint16(0xdffd):
            // This address is sometimes probed to see if speech roms are installed
            //log.Printf("R16 invalid address: $%.4X\n", addr)
        default:
            log.Panicf("R8 invalid address: $%.4X\n", addr)
    }
    return val
}

func (d *D8224Mem) W8(addr uint16, val uint8) {
    d.writes[addr] += 1
    switch {
        case addr >= 0x400 && addr <= 0x403:
            d.PIA.W8(addr-0x400, val)
        case d.valid[addr]:
            d.RxM[addr] = val
        case addr == uint16(0xfffe):
            // Playball wants to write to this address?
        default:
            log.Panicf("W8 invalid address: $%.4X\n", addr)
    }
    return
}

func (d *D8224Mem) R16(addr uint16) uint16 {
    var high, low uint8
    d.reads[addr] += 1
    d.reads[addr+1] += 1
    if d.valid[addr] && d.valid[addr+1] {
        high = d.RxM[addr]
        low  = d.RxM[addr+1]
    } else {
        if addr == uint16(0xeffd) {
            log.Printf("R16 invalid address: $%.4X\n", addr)
        } else {
            log.Panicf("R16 invalid address: $%.4X\n", addr)
        }
    }
    return (uint16(high)<<8) + uint16(low)
}

func (d *D8224Mem) W16(addr uint16, val uint16) {
    d.writes[addr] += 1
    d.writes[addr+1] += 1
    if addr <= 0x7E {
        d.RxM[addr] = uint8(val>>8)
        d.RxM[addr+1] = uint8(val)
    } else {
        // everything else is ROM
        log.Panicf("W16 invalid address: $%.4X\n", addr)
    }
    return
}
