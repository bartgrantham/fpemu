package cpu

import (
    "github.com/bartgrantham/fpemu/mem"
)

type interface CPU16 {
    Step(mmu mem.MMU) error
    Disasm(pc uint16, mmu mem.MMU16) (string, uint16) {
}
