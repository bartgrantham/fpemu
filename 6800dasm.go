package main

import (
    "fmt"
    "io"
    "io/ioutil"
    "os"
//    "strings"
)

type MemBuffer interface {
    U8(pc int)  byte
    U16(pc int) uint16
//    r32(pc int) uint32
//    r64(pc int) uint64
}

type M6800Mem []byte

func (m *M6800Mem) U8(addr int) byte {
    return (*m)[addr]
}
func (m *M6800Mem) U16(addr int) uint16 {
    return uint16((*m)[addr+1]) + ( uint16((*m)[addr])<<8 )
}

type Alignment int

const (
    Inh  Alignment  = iota  // inherent
    Rel  // relative
    Imb  // byte immediate
    Imw  // word immediate
    Dir  // direct (aka zero-page)
    Imd  // HD63701YO: immediate, direct
    Ext  // extended
    Idx  // x + byte offset
    Imx  // HD63701YO: immediate, x + byte offset
    Sx1  // HD63701YO, undocumented: byte from (s+1)
)

const STEP_OVER = 1<<16
const STEP_OUT  = 1<<17

type Opcode struct {
    Mnemonic     string
    AddrMode     Alignment
    InvalidMask  int  // invalid for 1:6800/6802/6808, 2:6801/6803, 4:HD63701
}

// 256 M680x opcodes + 4 alternate NSC-8105 opcodes
var M6800Ops [260]Opcode = [260]Opcode{
// 00
    {"ill", Inh, 7}, {"nop", Inh, 0}, {"ill", Inh, 7}, {"ill", Inh, 7},
    {"lsrd", Inh, 1}, {"asld", Inh, 1}, {"tap", Inh, 0}, {"tpa", Inh, 0},
    {"inx", Inh, 0}, {"dex", Inh, 0}, {"clv", Inh, 0}, {"sev", Inh, 0},
    {"clc", Inh, 0}, {"sec", Inh, 0}, {"cli", Inh, 0}, {"sei", Inh, 0},
// 10
    {"sba", Inh, 0}, {"cba", Inh, 0}, {"asx1", Sx1, 0}, {"asx2", Sx1, 0},
    {"ill", Inh, 7}, {"ill", Inh, 7}, {"tab", Inh, 0}, {"tba", Inh, 0},
    {"xgdx", Inh, 3}, {"daa", Inh, 0}, {"ill", Inh, 7}, {"aba", Inh, 0},
    {"ill", Inh, 7}, {"ill", Inh, 7}, {"ill", Inh, 7}, {"ill", Inh, 7},
// 20
    {"bra", Rel, 0}, {"brn", Rel, 0}, {"bhi", Rel, 0}, {"bls", Rel, 0},
    {"bcc", Rel, 0}, {"bcs", Rel, 0}, {"bne", Rel, 0}, {"beq", Rel, 0},
    {"bvc", Rel, 0}, {"bvs", Rel, 0}, {"bpl", Rel, 0}, {"bmi", Rel, 0},
    {"bge", Rel, 0}, {"blt", Rel, 0}, {"bgt", Rel, 0}, {"ble", Rel, 0},
// 30
    {"tsx", Inh, 0}, {"ins", Inh, 0}, {"pula", Inh, 0}, {"pulb", Inh, 0},
    {"des", Inh, 0}, {"txs", Inh, 0}, {"psha", Inh, 0}, {"pshb", Inh, 0},
    {"pulx", Inh, 1}, {"rts", Inh, 0}, {"abx", Inh, 1}, {"rti", Inh, 0},
    {"pshx", Inh, 1}, {"mul", Inh, 1}, {"wai", Inh, 0}, {"swi", Inh, 0},
// 40
    {"nega", Inh, 0}, {"ill", Inh, 7}, {"ill", Inh, 7}, {"coma", Inh, 0},
    {"lsra", Inh, 0}, {"ill", Inh, 7}, {"rora", Inh, 0}, {"asra", Inh, 0},
    {"asla", Inh, 0}, {"rola", Inh, 0}, {"deca", Inh, 0}, {"ill", Inh, 7},
    {"inca", Inh, 0}, {"tsta", Inh, 0}, {"ill", Inh, 7}, {"clra", Inh, 0},
// 50
    {"negb", Inh, 0}, {"ill", Inh, 7}, {"ill", Inh, 7}, {"comb", Inh, 0},
    {"lsrb", Inh, 0}, {"ill", Inh, 7}, {"rorb", Inh, 0}, {"asrb", Inh, 0},
    {"aslb", Inh, 0}, {"rolb", Inh, 0}, {"decb", Inh, 0}, {"ill", Inh, 7},
    {"incb", Inh, 0}, {"tstb", Inh, 0}, {"ill", Inh, 7}, {"clrb", Inh, 0},
// 60
    {"neg", Idx, 0}, {"aim", Imx, 3}, {"oim", Imx, 3}, {"com", Idx, 0},
    {"lsr", Idx, 0}, {"eim", Imx, 3}, {"ror", Idx, 0}, {"asr", Idx, 0},
    {"asl", Idx, 0}, {"rol", Idx, 0}, {"dec", Idx, 0}, {"tim", Imx, 3},
    {"inc", Idx, 0}, {"tst", Idx, 0}, {"jmp", Idx, 0}, {"clr", Idx, 0},
// 70
    {"neg", Ext, 0}, {"aim", Imd, 3}, {"oim", Imd, 3}, {"com", Ext, 0},
    {"lsr", Ext, 0}, {"eim", Imd, 3}, {"ror", Ext, 0}, {"asr", Ext, 0},
    {"asl", Ext, 0}, {"rol", Ext, 0}, {"dec", Ext, 0}, {"tim", Imd, 3},
    {"inc", Ext, 0}, {"tst", Ext, 0}, {"jmp", Ext, 0}, {"clr", Ext, 0},
// 80
    {"suba", Imb, 0}, {"cmpa", Imb, 0}, {"sbca", Imb, 0}, {"subd", Imw, 1},
    {"anda", Imb, 0}, {"bita", Imb, 0}, {"lda", Imb, 0}, {"sta", Imb, 0},
    {"eora", Imb, 0}, {"adca", Imb, 0}, {"ora", Imb, 0}, {"adda", Imb, 0},
    {"cmpx", Imw, 0}, {"bsr", Rel, 0}, {"lds", Imw, 0}, {"sts", Imw, 0},
// 90
    {"suba", Dir, 0}, {"cmpa", Dir, 0}, {"sbca", Dir, 0}, {"subd", Dir, 1},
    {"anda", Dir, 0}, {"bita", Dir, 0}, {"lda", Dir, 0}, {"sta", Dir, 0},
    {"eora", Dir, 0}, {"adca", Dir, 0}, {"ora", Dir, 0}, {"adda", Dir, 0},
    {"cmpx", Dir, 0}, {"jsr", Dir, 0}, {"lds", Dir, 0}, {"sts", Dir, 0},
// a0
    {"suba", Idx, 0}, {"cmpa", Idx, 0}, {"sbca", Idx, 0}, {"subd", Idx, 1},
    {"anda", Idx, 0}, {"bita", Idx, 0}, {"lda", Idx, 0}, {"sta", Idx, 0},
    {"eora", Idx, 0}, {"adca", Idx, 0}, {"ora", Idx, 0}, {"adda", Idx, 0},
    {"cmpx", Idx, 0}, {"jsr", Idx, 0}, {"lds", Idx, 0}, {"sts", Idx, 0},
// b0
    {"suba", Ext, 0}, {"cmpa", Ext, 0}, {"sbca", Ext, 0}, {"subd", Ext, 1},
    {"anda", Ext, 0}, {"bita", Ext, 0}, {"lda", Ext, 0}, {"sta", Ext, 0},
    {"eora", Ext, 0}, {"adca", Ext, 0}, {"ora", Ext, 0}, {"adda", Ext, 0},
    {"cmpx", Ext, 0}, {"jsr", Ext, 0}, {"lds", Ext, 0}, {"sts", Ext, 0},
// c0
    {"subb", Imb, 0}, {"cmpb", Imb, 0}, {"sbcb", Imb, 0}, {"addd", Imw, 1},
    {"andb", Imb, 0}, {"bitb", Imb, 0}, {"ldb", Imb, 0}, {"stb", Imb, 0},
    {"eorb", Imb, 0}, {"adcb", Imb, 0}, {"orb", Imb, 0}, {"addb", Imb, 0},
    {"ldd", Imw, 1}, {"_std", Imw, 1}, {"ldx", Imw, 0}, {"stx", Imw, 0},
// d0
    {"subb", Dir, 0}, {"cmpb", Dir, 0}, {"sbcb", Dir, 0}, {"addd", Dir, 1},
    {"andb", Dir, 0}, {"bitb", Dir, 0}, {"ldb", Dir, 0}, {"stb", Dir, 0},
    {"eorb", Dir, 0}, {"adcb", Dir, 0}, {"orb", Dir, 0}, {"addb", Dir, 0},
    {"ldd", Dir, 1}, {"_std", Dir, 1}, {"ldx", Dir, 0}, {"stx", Dir, 0},
// e0
    {"subb", Idx, 0}, {"cmpb", Idx, 0}, {"sbcb", Idx, 0}, {"addd", Idx, 1},
    {"andb", Idx, 0}, {"bitb", Idx, 0}, {"ldb", Idx, 0}, {"stb", Idx, 0},
    {"eorb", Idx, 0}, {"adcb", Idx, 0}, {"orb", Idx, 0}, {"addb", Idx, 0},
    {"ldd", Idx, 1}, {"_std", Idx, 1}, {"ldx", Idx, 0}, {"stx", Idx, 0},
// f0
    {"subb", Ext, 0}, {"cmpb", Ext, 0}, {"sbcb", Ext, 0}, {"addd", Ext, 1},
    {"andb", Ext, 0}, {"bitb", Ext, 0}, {"ldb", Ext, 0}, {"stb", Ext, 0},
    {"eorb", Ext, 0}, {"adcb", Ext, 0}, {"orb", Ext, 0}, {"addb", Ext, 0},
    {"ldd", Ext, 1}, {"_std", Ext, 1}, {"ldx", Ext, 0}, {"stx", Ext, 0},
// NSC-8105 alternate instructions: 0xfc, 0xec, 0xbb, 0xb2
    {"addx", Ext, 0}, {"adcx", Imb, 0}, {"bitx", Imx, 0}, {"stx", Imx, 0},
}

func disassemble(w io.Writer, pc int, mem MemBuffer, ram MemBuffer) int {
    flags := 0
    advance := 1
    invalid_mask := 1  // 6800/6802/6808/8105==1, 6801/6803==2, default=4
    code := int(mem.U8(pc))
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
    switch opcode.Mnemonic {
        case "bsr", "jsr":
            flags = STEP_OVER
        case "rti", "rts":
            flags = STEP_OUT
    }

    if (invalid_mask & opcode.InvalidMask) != 0 {
        fields = append(fields, "ILL")
        output := fmt.Sprintf("%s    %-8s  %s\n", fields[0], fields[1], fields[2])
        w.Write([]byte(output))
        _ = flags
        return 1 // | flags // | SUPPORTED
    }
    desc := fmt.Sprintf("%-5s ", opcode.Mnemonic)
    switch opcode.AddrMode {
        case Rel:  // relative
            // "$%04X", pc + (int8_t)params.r8(pc+1) + 2
            fields[1] += fmt.Sprintf("%.2X", mem.U8(pc+1))
            offset := mem.U8(pc+1)
//            if (offset & 0x80) == 0x80 {
//                offset -= 128
//            }
// XXX: may be incorrect
            desc += fmt.Sprintf("$%04X", pc + int(offset + 2))
            advance += 1
        case Imb:  // byte immediate
            // "#$%02X", params.r8(pc+1)
            fields[1] += fmt.Sprintf("%.2X", mem.U8(pc+1))
            desc += fmt.Sprintf("0x%02X", mem.U8(pc+1))
            advance += 1
        case Imw:  // word immediate
            // "#$%04X", params.r16(pc+1)
            fields[1] += fmt.Sprintf("%.2X%.2X", mem.U8(pc+1), mem.U8(pc+2))
            desc += fmt.Sprintf("$%04X", mem.U16(pc+1))
            advance += 2
        case Idx:  // x + byte offset
            // "(x+$%02X)", params.r8(pc+1)
            fields[1] += fmt.Sprintf("%.2X", mem.U8(pc+1))
            desc += fmt.Sprintf("(x+0x%02X)", mem.U8(pc+1))
            advance += 1
        case Imx:  // HD63701YO: immediate, x + byte offset
            // "#$%02X,(x+$%02x)", params.r8(pc+1), params.r8(pc+2)
            fields[1] += fmt.Sprintf("%.2X%.2X", mem.U8(pc+1), mem.U8(pc+2))
            desc += fmt.Sprintf("0x%02X,(x+0x%02x)", mem.U8(pc+1), mem.U8(pc+2))
            advance += 2
        case Dir:  // direct (aka zero-page)
            // "$%02X", params.r8(pc+1)
            fields[1] += fmt.Sprintf("%.2X", mem.U8(pc+1))
            desc += fmt.Sprintf("0x%02X", mem.U8(pc+1))
            advance += 1
        case Imd:  // HD63701YO: immediate, direct address
            // "#$%02X,$%02X", params.r8(pc+1), params.r8(pc+2)
            fields[1] += fmt.Sprintf("%.2X%.2X", mem.U8(pc+1), mem.U8(pc+2))
            desc += fmt.Sprintf("0x%02X,0x%02X", mem.U8(pc+1), mem.U8(pc+2))
            advance += 2
        case Ext:  // extended
            // "$%04X", params.r16(pc+1)
            fields[1] += fmt.Sprintf("%.2X%.2X", mem.U8(pc+1), mem.U8(pc+2))
            desc += fmt.Sprintf("$%04X", mem.U16(pc+1))
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

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: disasm somerom.rom")
        os.Exit(-1)
    }
    romfile, err := os.Open(os.Args[1]);
    if err != nil {
        fmt.Println("Error opening '" + os.Args[1] + "':", err.Error())
        os.Exit(-1)
    }
    defer romfile.Close()

    buf, err := ioutil.ReadAll(romfile)
    if err != nil {
        fmt.Println("Error reading '" + os.Args[1] + "':", err.Error())
        os.Exit(-1)
    }

    rom := M6800Mem(buf)

    pc := 0
    for {
        advance := disassemble(os.Stdout, pc, MemBuffer(&rom), MemBuffer(&rom))
        pc += advance
        if pc >= len(rom) {
            break
        }
    }
}
