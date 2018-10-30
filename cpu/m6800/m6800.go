package m6800

import (
//    "encoding/binary"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/bartgrantham/fpemu/mem"
    "github.com/bartgrantham/fpemu/pia"
    "github.com/bartgrantham/fpemu/pia/m6821"
    "github.com/bartgrantham/fpemu/ui"

    "github.com/gdamore/tcell"
)

var crystal float32 = 3580000.0 / 4

var wtflog *os.File
var wavout *os.File

func init() {
   wtflog, _ = os.OpenFile("WTF.log", os.O_RDWR|os.O_CREATE, 0755)
   wavout, _ = os.OpenFile("out.f32", os.O_RDWR|os.O_CREATE, 0755)
}

type M6800 struct {
    PC      uint16
    X       uint16
    A       uint8
    B       uint8
    CC      uint8
    SP      uint16
    PIA     pia.PIA
    NMI     bool
//    cps     float64
//    cycles  float64
}

const (
    C   uint8  = 1 << iota  // carry
    V   // overflow
    Z   // zero
    N   // negative / sign
    I   // interrupt masked, set by hardware interrupt or SEI opcode
    H   // half-carry
)

func NewM6800(mmu mem.MMU16, pia pia.PIA) *M6800{
    m := M6800{PC:mmu.R16(0xFFFE), PIA:pia}
//    m.SEI_0f(mmu)  // CPU starts with interrupts disabled/masked
    return &m
}

func (m *M6800) Status() string {
    fmtstr := "PC: $%.4X ($%.4X) ; X:$%.4X ; A:0x%.2X ; B:0x%.2X ; CC:0x%08b ; SP:$%.4X"
    return fmt.Sprintf(fmtstr, m.PC, m.PC, m.X, m.A, m.B, m.CC, m.SP)
}

func (m *M6800) Step(mmu mem.MMU16) (int, error) {
    //ui.Log(m.Status())
    if (m.PIA.IRQ(0) || m.PIA.IRQ(1)) && (m.CC & I != I) {
        m.save_registers(mmu)
        //m.SEI_0f(mmu)  // interrupts masked
        SEI_0f(m, mmu)  // interrupts masked
        if m.NMI {
            m.PC = mmu.R16(0xFFFC)
        } else {
            m.PC = mmu.R16(0xFFF8)
        }
    }
    opcode := mmu.R8(m.PC)
    m.PC += 1
    count, err := m.dispatch(opcode, mmu)
/*
    fmt.Fprintln(wtflog, "", m.Status())
    first32 := "    "
    for addr:=uint16(0); addr<0x20; addr++ {
        val, _, _ := mmu.Peek8(addr)
        first32 += fmt.Sprintf("%.2x ", val)
        if addr % 4 == 3 {
            first32 += " "
        }
    }
    fmt.Fprintln(wtflog, first32)
*/
    return count, err
}

/*
* When RTI is executed the saved registers are restored and processing proceeds from the interrupted point
* On IRQ:
    * If I=0 and the IRQ line goes low for at least one ϕ2 cycle
    * registers CC, B, A, X, PC (7 bytes, in that order, big endian for X and PC) are stored at SP-6..SP
    * I is set to 1 (IRQs masked)
    * 16-bit (big endian) irq vector is loaded from $FFF8 and the irq begins processing
* On NMI:
    * If the NMI line goes low for at least one ϕ2 cycle
    * registers CC, B, A, X, PC (7 bytes, in that order, big endian for X and PC) are stored at SP-6..SP
    * I is set to 1 (IRQs masked)
    * 16-bit (big endian) irq vector is loaded from $FFFC and the irq begins processing
* On SWI:
    * registers CC, B, A, X, PC (7 bytes, in that order, big endian for X and PC) are stored at SP-6..SP
    * I is set to 1 (IRQs masked)
    * 16-bit (big endian) irq vector is loaded from $FFFA and the irq begins processing

On firepower port A is the DAC, port B is from the mainboard
*/

func (m *M6800) Callback(mmu mem.MMU16, ctrl chan rune, pia *m6821.M6821) func([]float32) {
    var code rune
// this calculation is probably all wrong...
    hostrate := float32(44100)
    cycles_per_sample := crystal / hostrate
log.Printf("crystal %.8f, cps %.8f\n", crystal, cycles_per_sample)
    var jitter, samp float32
    var i, total_cycles int
    return func(out []float32) {
        total_cycles = 0
        start := time.Now()
        for i=0; i<len(out); i+=2 {
            select {
                case code = <-ctrl:
                    char := uint8(code)
                    if char >= '0' && char <= 'O' {
                        m.PIA.Write(1, (char-0x10)^0xFF)  // 0x30 == '0' -> 00-1f
                    }
                default:
            }
            for jitter < 0 {
                cycles, _ := m.Step(mmu)
                jitter += float32(cycles)
                total_cycles += cycles
            }
            samp = (float32(pia.ORA) / 256) - .5
            samp += pia.CVSD.State * 2
            out[i] = samp
            out[i+1] = samp
            jitter -= cycles_per_sample
        }
        wav := make([]int16, len(out))
        max := float32(-1.0)
        min := float32(1.0)
        for i, s := range out {
            if s < min {
                min = s
            }
            if s > max {
                max = s
            }
            wav[i] = int16(s * 24000)
        }
        //binary.Write(wavout, binary.LittleEndian, wav)
        ui.Log(fmt.Sprintf("%dcyc, %dsamp in %v, jitter %.4f, %.3f..%.3f", total_cycles, len(out) / 2, time.Since(start), jitter, min, max))
    }
}
/*
func (m *M6800) Callback(mmu mem.MMU16, ctrl chan rune, pia *m6821.M6821) func([]float32) {
    var chr rune
// this calculation is probably all wrong...
    audiorate := float32(44100)
    cycles_per_sample := crystal / audiorate
log.Printf("crystal %.8f, cps %.8f\n", crystal, cycles_per_sample)
    var total_cycles, elapsed, samp float32
    var i int
    return func(out []float32) {
        total_cycles = 0
        start := time.Now()
        for i=0; i<len(out); i+=2 {
            select {
                case chr = <-ctrl:
                    if chr >= '0' && chr <= 'o' {
                        m.PIA.Write(1, uint8(chr-'0'))
                    }
                default:
            }
            cycles, _ := m.Step(mmu)
            elapsed += float32(cycles)
            total_cycles += elapsed
            if elapsed > cycles_per_sample {
                samp = float32(pia.ORA) / float32(256)
                out[i] = samp
                out[i+1] = samp
                elapsed -= cycles_per_sample
                continue
            }
        }
        ui.Log(fmt.Sprintf("%.0f cycles for %d samples in %v", total_cycles, len(out) / 2, time.Since(start)))
    }
}
*/

func (m *M6800) Run(mmu mem.MMU16, ctrl chan rune, screen tcell.Screen) {
    var chr rune
    var tick *time.Ticker
    rate := float32(100)
    cycles_per_rate := crystal / rate
    tick = time.NewTicker(time.Duration(float32(time.Second)/rate))
    _ = tick
    var total, remainder float32
    go func() {
/*
        defer func() {
            //if r := recover(); r != nil {
            //}
            screen.Fini()
            ui.DumpLog()
            for addr:=0; addr<0x80; addr++ {
                if addr & 0x0F == 0x00 {
                    fmt.Printf("$%.2x:  ", addr)
                }
                val, _, _ := mmu.Peek8(uint16(addr))
                fmt.Printf("%.2x  ", val)
                if addr & 0x0F == 0x0F {
                    fmt.Println()
                }
            }
        }()
*/
        for {
            select {
                case chr = <-ctrl:
                    if chr >= '0' && chr <= 'o' {
                        m.PIA.Write(1, uint8(chr-'0'))
                    }
                case <-tick.C:
                    total = remainder
//                    start := time.Now()
                    // run one "rate" worth of cycles
                    for {
                        if total > cycles_per_rate {
                            remainder = total - cycles_per_rate
                            break
                        }
                        cycles, _ := m.Step(mmu)
                        total += float32(cycles)
                    }
//                    ui.Log(fmt.Sprintf("%f cycles in %s", total, time.Since(start)))
//                default:
//                    m.Step(mmu)
            }
        }
    }()
}
