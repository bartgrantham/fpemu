package pia

type PIA interface {
    R8(addr uint16) uint8
    W8(addr uint16, val uint8)
    Read(port uint16) uint8
    Write(port uint16, val uint8)
    IRQ(line uint16) bool
    Reset()
}
