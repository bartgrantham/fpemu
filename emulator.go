package main

import (
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "time"

    "github.com/bartgrantham/fpemu/cpu/m6800"
    "github.com/bartgrantham/fpemu/mem"
    "github.com/bartgrantham/fpemu/mem/firepower"
    "github.com/bartgrantham/fpemu/pia/m6821"
    "github.com/bartgrantham/fpemu/ui"
    "github.com/gdamore/tcell"
)

/*
    draw disasm
    load queue of commands from cli
    keys for: pause, step, 1x .1s .01s .001s
    draw PIA output
*/

func main() {
    if len(os.Args) < 5 {
        fmt.Println("Usage: fpemu V_IC7.532 V_IC5.532 V_IC6.532 SOUND3.716")
        os.Exit(-1)
    }
    f1, _ := os.Open(os.Args[1])
    f2, _ := os.Open(os.Args[2])
    f3, _ := os.Open(os.Args[3])
    f4, _ := os.Open(os.Args[4])

    ic7, _ := ioutil.ReadAll(f1)
    ic5, _ := ioutil.ReadAll(f2)
    ic6, _ := ioutil.ReadAll(f3)
    ic12, _ := ioutil.ReadAll(f4)

    // Init emulation
    pia := m6821.M6821{}
    mmu := firepower.NewFirepowerMem(ic5, ic7, ic6, ic12, &pia)
    m6800 := m6800.NewM6800(mmu, &pia)

/*
Audio:

* remember: the callback will consume the entire buffer in a single blast
    * doubling of missing samples will result in a single sample correct followed by a hundred incorrect ones
    *
*/
    // Init audio
    // Init PIA->DAC
    m6821_OUTA := make(chan uint8, 1024)
    in_rate := float32(8000)
    in_rate_ns := time.Duration(float32(time.Second)/in_rate)
    
    go func() {
        var tick *time.Ticker
        tick = time.NewTicker(in_rate_ns)
        for {
            <-tick.C
            m6821_OUTA <- pia.ORA & pia.DDRA
//            m6821_OUTA <- 0
//            m6821_OUTA <- 255
        }
    }()

    time.Sleep(100 * time.Millisecond)
    out_rate := float32(44100)
    in_per_out := in_rate / out_rate
    callback := func() func([]float32) {
        var samp float32
        var tick float32
        var i int
        var piaout uint8
        return func (out []float32) {
            for i=0; i<len(out); i+=2 {
                out[i] = samp
                out[i+1] = samp
                tick += in_per_out
                if tick > 1 {
                    select {
                        case piaout = <-m6821_OUTA:
                            samp = float32(piaout) / float32(256)
                        default:
                            // underflow? use the last sample, I guess
                    }
                    tick -= 1
                }
            }
        }
    }()
    err := ui.StartAudio(callback)
/*
    err := ui.StartAudio(func (out []float32){
        var left, right float32
        for i := range out {
            select {
                default:
                    if i < 2 {
                    
                    } else {
                    }
            }
        
            out[i] = float32(<- m6821_OUTA) / float32(256)
            //out[i] = float32(i) / float32(len(out))
        }
    })
*/
    if err != nil {
        fmt.Println("Couldn't start audio:", err)
        os.Exit(-1)
    }
    defer ui.StopAudio()

    // Init screen
    screen, err := tcell.NewScreen()
    if err != nil {
        fmt.Println("Error opening screen:", err)
        os.Exit(-1)
    }
    if err := screen.Init(); err != nil {
        fmt.Println("Error opening screen:", err)
        os.Exit(-1)
    }
    defer func() {
        if r := recover(); r != nil {
            screen.Fini()
            log.Println("Error:", r)
            os.Exit(-1)
        }
    }()

    // Run UI
    dl := []ui.Draw{func(){
        ramBox(screen, 3, 1, "IRAM", 0x0, mmu)
//        ramBox(screen, 3, 13, "ORAM", 0x400, mmu)
        cpuBox(screen, 64, 1, m6800)
//        mmu.ClearPeekCounts()
        ui.LogBox(screen, 3, 13, "Log")
    }}
    tui := ui.TextUI{
        Screen:screen,
        Tick:time.NewTicker(25 * time.Millisecond),
        DisplayList:dl,
    }
    tui.Run()


    // Run emulation
    ctrl := make(chan rune, 10)
    m6800.Run(mmu, ctrl)

    evtloop:
    for {
        evt := screen.PollEvent()
        e, ok := evt.(*tcell.EventKey)
        if ! ok {
            continue
        }
        switch e.Key() {
            case tcell.KeyCtrlC:
                break evtloop
            case tcell.KeyRune:
                chr := e.Rune()
                if chr >= 'a' && chr <= 'p' {
                    ctrl <- chr
                }
            default:
                ctrl <- '_'
        }
    }
    screen.Fini()
    ui.DumpLog()
}

// draw 128 bytes from ram
func ramBox(s tcell.Screen, x, y int, label string, addr uint16, mem mem.MMU16) {
    ui.Box(s, x, y, 57, 11)
    style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Underline(true)
    colhead := "x0 x1 x2 x3 x4 x5 x6 x7  x8 x9 xA xB xC xD xE xF"
    ui.DrawString(s, x+8, y+2, style, colhead)
    style = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
    ui.DrawString(s, x+6, y, style, " "+label+" ")

    row := 0
    for {
        style = tcell.StyleDefault.Foreground(tcell.ColorWhite)
        low := int(addr&0xF)
        if low == 0 {
            rowhead := fmt.Sprintf("$%.4X", addr&0xFFF0)
            ui.DrawString(s, x+2, row+y+3, style, rowhead)
        }
        col := x + (low*3) + 4
        if low >= 8 {
            col += 1
        }
        val, reads, writes := mem.Peek8(addr)
        switch {
            case writes==0 && reads>0:
                style = tcell.StyleDefault.Foreground(tcell.ColorGreen)
            case reads==0 && writes>0:
                style = tcell.StyleDefault.Foreground(tcell.ColorRed)
            case reads>0 && writes>0:
                style = tcell.StyleDefault.Foreground(tcell.ColorYellow)
            default:
                style = tcell.StyleDefault.Foreground(tcell.ColorGray)
        }
        cell := fmt.Sprintf("%.2X", val)
        ui.DrawString(s, col+4, row+y+3, style, cell)
        addr += 1
        if (addr % 16) == 0 {
            row += 1
        }
        if row >= 8 {
            break
        }
    }
}


func cpuBox(s tcell.Screen, x, y int, cpu *m6800.M6800) {
    ui.Box(s, x, y, 20, 11)
    style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
    ui.DrawString(s, x+2, y, style, " M6802 ")
    style = tcell.StyleDefault.Foreground(tcell.ColorGray)
    col := x+2
    row := y+2
    ui.DrawString(s, col, row, style, "PC:")
    ui.DrawString(s, col, row+1, style, "SP:")
    ui.DrawString(s, col, row+2, style, " X:")
    ui.DrawString(s, col, row+3, style, " A:")
    ui.DrawString(s, col, row+4, style, " B:")
    ui.DrawString(s, col, row+6, style, "      --HINZVC")
    ui.DrawString(s, col, row+7, style, "CC:")
    style = tcell.StyleDefault.Foreground(tcell.ColorWhite)
    col = col + 4
    ui.DrawString(s, col, row, style, fmt.Sprintf("$%.4X", cpu.PC))
    ui.DrawString(s, col, row+1, style, fmt.Sprintf("$%.4X", cpu.SP))
    ui.DrawString(s, col, row+2, style, fmt.Sprintf("$%.4X", cpu.X))
    ui.DrawString(s, col, row+3, style, fmt.Sprintf("0x%.2X", cpu.A))
    ui.DrawString(s, col, row+4, style, fmt.Sprintf("0x%.2X", cpu.B))
    ui.DrawString(s, col, row+7, style, fmt.Sprintf("0b%.8b", cpu.CC))
}
