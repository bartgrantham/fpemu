package cpu

import (
    "github.com/bartgrantham/fpemu/mem"
)

type interface CPU {
    Step(mmu mem.MMU) error
    Disasm(pc uint16)  uint16
}
