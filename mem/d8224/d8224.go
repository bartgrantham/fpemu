package d8224

import (
    "fmt"
    "log"
    "github.com/bartgrantham/fpemu/pia"
)

type D8224Mem struct {
    PIA       pia.PIA
    RxM       [1<<16]uint8  // $0000-$FFFF
    validr    [1<<16]bool
    validw    [1<<16]bool
    reads     [1<<16]int
    writes    [1<<16]int
    fastfail  bool
}

type reads struct {
}

func NewD8224Mem(pia pia.PIA) *D8224Mem {
    d := D8224Mem{PIA:pia}
    // the 6802 has 128b built-in
    for i:=0; i<128; i++ {
        d.validr[i] = true
        d.validw[i] = true
    }
    d.fastfail = true
    return &d
}

func (d *D8224Mem) Mount(addr uint16, data []byte, rw bool) error {
    if int(addr) + len(data) > 1<<16 {
        return fmt.Errorf("invalid mount")
    }
    if int(addr) < 128 {
        return fmt.Errorf("invalid mount")
    }
    for i, b := range data {
        d.RxM[int(addr) + i] = b
        d.validr[int(addr) + i] = true
        if rw {
            d.validw[int(addr) + i] = true
        }
    }
    return nil
}

// temporary
func (d *D8224Mem) String() string {
    return d.PIA.String()
}

func (d *D8224Mem) Valid(addr uint16) (bool, bool) {
    return d.validr[int(addr)], d.validw[int(addr)]
}

func (d *D8224Mem) Peek8(addr uint16) uint8 {
    val := uint8(0)
    switch {
        case addr >= 0x400 && addr <= 0x403:
            val = d.PIA.R8(addr-0x400)
        case d.validr[addr]:
            val = d.RxM[addr]
        default:
            err := fmt.Sprintf("Peek8 invalid address: $%.4X", addr)
            if d.fastfail {  panic(err)  } else {  log.Println(err)  }
    }
    return val
}

func (d *D8224Mem) R8(addr uint16) uint8 {
    d.reads[addr] += 1
    val := uint8(0)
    switch {
        case addr >= 0x400 && addr <= 0x403:
            val = d.PIA.R8(addr-0x400)
        case d.validr[addr]:
            val = d.RxM[addr]
//        case addr == uint16(0xeffd) || addr == uint16(0xdffd):
            // This address is sometimes probed to see if speech roms are installed
            //log.Printf("R16 invalid address: $%.4X", addr)
        default:
            err := fmt.Sprintf("R8 invalid address: $%.4X", addr)
            if d.fastfail {  panic(err)  } else {  log.Println(err)  }
    }
    return val
}

func (d *D8224Mem) W8(addr uint16, val uint8) {
    d.writes[addr] += 1
    switch {
        case addr >= 0x400 && addr <= 0x403:
            d.PIA.W8(addr-0x400, val)
        case d.validw[addr]:
            d.RxM[addr] = val
        default:
            err := fmt.Sprintf("W8 invalid address (val): $%.4X (%.2X)", addr, val)
            if d.fastfail {  panic(err)  } else {  log.Println(err)  }
    }
    return
}

func (d *D8224Mem) R16(addr uint16) uint16 {
    var high, low uint8
    d.reads[addr] += 1
    d.reads[addr+1] += 1
    if d.validr[addr] && d.validr[addr+1] {
        high = d.RxM[addr]
        low  = d.RxM[addr+1]
    } else {
        err := fmt.Sprintf("R16 invalid address: $%.4X", addr)
        if d.fastfail {  panic(err)  } else {  log.Println(err)  }
    }
    return (uint16(high)<<8) + uint16(low)
}

func (d *D8224Mem) W16(addr uint16, val uint16) {
    d.writes[addr] += 1
    d.writes[addr+1] += 1
    if d.validw[addr] && d.validw[addr+1] {
        d.RxM[addr] = uint8(val>>8)
        d.RxM[addr+1] = uint8(val)
    } else {
        err := fmt.Sprintf("W16 invalid address: $%.4X", addr)
        if d.fastfail {  panic(err)  } else {  log.Println(err)  }
    }
    return
}

func (d *D8224Mem) Heat(start, end uint16) ([]uint8, []int, []int) {
    if end <= start {
        return nil, nil, nil
    }
    for i:=start; i<end; i++ {
        if d.reads[i] > 0 {
            d.reads[i] /= 2
        }
        if d.writes[i] > 0 {
            d.writes[i] /= 2
        }
    }
    return d.RxM[start:end], d.reads[start:end], d.writes[start:end]
}

