package m6800

import (
    "io"
    "fmt"

    "github.com/bartgrantham/fpemu/mem"
)

func (c *M6800) Disasm(w io.Writer, pc uint16, mmu mem.MMU16) uint16 {
    advance := uint16(1)
    invalid_mask := 1  // 6800/6802/6808/8105==1, 6801/6803==2, default=4
    code := int(mmu.R8(pc))
    fields := []string{}

    fields = append(fields, fmt.Sprintf("0x%.4X", pc))
    fields = append(fields, fmt.Sprintf("%.2X", code))
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
        fields = append(fields, "ILL")
        output := fmt.Sprintf("%s    %-8s  %s\n", fields[0], fields[1], fields[2])
        w.Write([]byte(output))
        return 1
    }
    desc := fmt.Sprintf("%-5s ", opcode.Mnemonic)
    switch opcode.AddrMode {
        case Rel:  // relative
            // "$%04X", pc + (int8_t)params.r8(pc+1) + 2
            fields[1] += fmt.Sprintf("%.2X", mmu.R8(pc+1))
            offset := int8(mmu.R8(pc+1))
            // XXX: may be incorrect
            calc := ""
            if offset < 0 {
                calc = fmt.Sprintf("$%04X + 2 %d", pc, offset)
            } else {
                calc = fmt.Sprintf("$%04X + 2 + %d", pc, offset)
            }
            desc += fmt.Sprintf("$%04X (%s)", int32(pc) + 2 + int32(offset), calc)
            advance += 1
        case Imb:  // byte immediate
            // "#$%02X", params.r8(pc+1)
            fields[1] += fmt.Sprintf("%.2X", mmu.R8(pc+1))
            desc += fmt.Sprintf("0x%02X", mmu.R8(pc+1))
            advance += 1
        case Imw:  // word immediate
            // "#$%04X", params.r16(pc+1)
            fields[1] += fmt.Sprintf("%.2X%.2X", mmu.R8(pc+1), mmu.R8(pc+2))
            desc += fmt.Sprintf("$%04X", mmu.R16(pc+1))
            advance += 2
        case Idx:  // x + byte offset
            // "(x+$%02X)", params.r8(pc+1)
            fields[1] += fmt.Sprintf("%.2X", mmu.R8(pc+1))
            desc += fmt.Sprintf("(x+0x%02X)", mmu.R8(pc+1))
            advance += 1
        case Imx:  // HD63701YO: immediate, x + byte offset
            // "#$%02X,(x+$%02x)", params.r8(pc+1), params.r8(pc+2)
            fields[1] += fmt.Sprintf("%.2X%.2X", mmu.R8(pc+1), mmu.R8(pc+2))
            desc += fmt.Sprintf("0x%02X,(x+0x%02x)", mmu.R8(pc+1), mmu.R8(pc+2))
            advance += 2
        case Dir:  // direct (aka zero-page)
            // "$%02X", params.r8(pc+1)
            fields[1] += fmt.Sprintf("%.2X", mmu.R8(pc+1))
            desc += fmt.Sprintf("0x%02X", mmu.R8(pc+1))
            advance += 1
        case Imd:  // HD63701YO: immediate, direct address
            // "#$%02X,$%02X", params.r8(pc+1), params.r8(pc+2)
            fields[1] += fmt.Sprintf("%.2X%.2X", mmu.R8(pc+1), mmu.R8(pc+2))
            desc += fmt.Sprintf("0x%02X,0x%02X", mmu.R8(pc+1), mmu.R8(pc+2))
            advance += 2
        case Ext:  // extended
            // "$%04X", params.r16(pc+1)
            fields[1] += fmt.Sprintf("%.2X%.2X", mmu.R8(pc+1), mmu.R8(pc+2))
            desc += fmt.Sprintf("$%04X", mmu.R16(pc+1))
            advance += 2
        case Sx1:  // HD63701YO, undocumented: byte from (s+1)
            // util::stream_format(stream, "(s+1)")
        case Inh:  // no params
    }
    fields = append(fields, desc)
    output := fmt.Sprintf("%s    %-8s  %s\n", fields[0], fields[1], fields[2])
    w.Write([]byte(output))
    return advance
}
