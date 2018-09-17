package mem

type MMU16 interface {
    R8(addr uint16) uint8
    W8(addr uint16, val uint8)
    R16(addr uint16) uint16
//    W16(addr uint16, val uint16) uint16
}
