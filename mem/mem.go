package mem

// it'd be nice to have proper error types, but the use of
// the R/W methods inlined in expressions like:
//     mmu.R8(uint16(mmu.R8(m.PC))) 
// is too nice to give up

type MMU16 interface {
    R8(addr uint16) uint8
    W8(addr uint16, val uint8)
    R16(addr uint16) uint16
    W16(addr uint16, val uint16)
    Peek8(addr uint16) uint8
    Valid(addr uint16) (read bool, write bool)
    Heat(start, end uint16) (vals []uint8, reads []int, writes []int)
    String() string  //temporary
}

