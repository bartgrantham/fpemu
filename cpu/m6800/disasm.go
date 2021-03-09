package m6800

import (
    "fmt"

    "github.com/bartgrantham/fpemu/mem"
)

func (c *M6800) Disasm(pc uint16, mmu mem.MMU16) (output string, advance uint16) {
    defer func() {
        if r := recover(); r != nil {
            advance = 0
            output = fmt.Sprintf("0x%.4X    --  ILL", pc)
        }
    }()

    var instbytes, desc string
    var invalid_mask, code int

    advance = 1
    invalid_mask = 1  // 6800/6802/6808/8105==1, 6801/6803==2, default=4
    code = int(mmu.R8(pc))

    instbytes = fmt.Sprintf("%.2X", code)
    if false {  // NSC-8105 == true
        // swap bits
        code = (code & 0x3c) | ((code & 0x41) << 1) | ((code & 0x82) >> 1)
        switch code {
            case 0xfc: code = 0x0100;
            case 0xec: code = 0x0101;
            case 0x7b: code = 0x0102;
            case 0x71: code = 0x0103;
        }
    }
    opcode := M6800Ops[code]
    if (invalid_mask & opcode.InvalidMask) != 0 {
        return fmt.Sprintf("0x%.4X    --  ILL", pc), 1
    }
    desc = fmt.Sprintf("%-5s ", opcode.Mnemonic)
    switch opcode.AddrMode {
        case Rel:  // relative
            instbytes += fmt.Sprintf("%.2X", mmu.R8(pc+1))
            offset := int8(mmu.R8(pc+1))
            desc += fmt.Sprintf("$%04X", int32(pc) + 2 + int32(offset))
            if offset < 0 {
                desc += " ("+ fmt.Sprintf("$%04X+2 - %d", pc, -offset) +")"
            } else {
                desc += " ("+ fmt.Sprintf("$%04X+2 + %d", pc, offset) +")"
            }
            advance += 1

        case Imb:  // byte immediate
            instbytes += fmt.Sprintf("%.2X", mmu.R8(pc+1))
            desc += fmt.Sprintf("0x%02X", mmu.R8(pc+1))
            advance += 1

        case Imw:  // word immediate
            instbytes += fmt.Sprintf("%.2X%.2X", mmu.R8(pc+1), mmu.R8(pc+2))
            desc += fmt.Sprintf("$%04X", mmu.R16(pc+1))
            advance += 2

        case Idx:  // X + byte offset
            instbytes += fmt.Sprintf("%.2X", mmu.R8(pc+1))
            desc += fmt.Sprintf("(x+0x%02X)", mmu.R8(pc+1))
            advance += 1

        case Imx:  // HD63701YO: immediate, X + byte offset
            instbytes += fmt.Sprintf("%.2X%.2X", mmu.R8(pc+1), mmu.R8(pc+2))
            desc += fmt.Sprintf("0x%02X,(x+0x%02x)", mmu.R8(pc+1), mmu.R8(pc+2))
            advance += 2

        case Dir:  // direct (aka zero-page)
            instbytes += fmt.Sprintf("%.2X", mmu.R8(pc+1))
            desc += fmt.Sprintf("0x%02X", mmu.R8(pc+1))
            advance += 1

        case Imd:  // HD63701YO: immediate, direct address
            instbytes += fmt.Sprintf("%.2X%.2X", mmu.R8(pc+1), mmu.R8(pc+2))
            desc += fmt.Sprintf("0x%02X,0x%02X", mmu.R8(pc+1), mmu.R8(pc+2))
            advance += 2

        case Ext:  // extended
            instbytes += fmt.Sprintf("%.2X%.2X", mmu.R8(pc+1), mmu.R8(pc+2))
            desc += fmt.Sprintf("$%04X", mmu.R16(pc+1))
            advance += 2

        case Sx1:  // HD63701YO, undocumented: byte from (s+1)

        case Inh:  // no params
    }
    output = fmt.Sprintf("0x%.4X    %-8s  %s", pc, instbytes, desc)
    return output, advance
}
