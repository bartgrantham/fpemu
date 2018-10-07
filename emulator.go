package main

import (
    "fmt"
    "io/ioutil"
    "log"
    "os"
//    "time"

    "github.com/bartgrantham/fpemu/cpu/m6800"
    "github.com/bartgrantham/fpemu/mem"
    "github.com/bartgrantham/fpemu/mem/firepower"
    "github.com/gdamore/tcell"
)

/*
    draw disasm
    draw log!
    accept keystroke runcommands
    load queue of commands from cli
    free-run CPU
*/

func drawString(s tcell.Screen, x, y int, style tcell.Style, str string) {
    for _, c := range str {
        s.SetContent(x, y, c, []rune{}, style)
        x += 1
    }
}

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

    mmu := firepower.NewFirepowerMem(ic5, ic7, ic6, ic12)
    m6800 := m6800.NewM6800(mmu)

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

    ctrl := make(chan string, 10)
    m6800.Run(mmu, ctrl)

    evtloop:
    for {
        ramBox(screen, 3, 1, "IRAM", 0x0, mmu)
        ramBox(screen, 3, 13, "ORAM", 0x400, mmu)
        cpuBox(screen, 64, 1, m6800)
        screen.Show()
        mmu.ClearPeekCounts()
        e := screen.PollEvent()
        switch e := e.(type) {
            case *tcell.EventKey:
                if e.Key() == tcell.KeyCtrlC {
                    break evtloop
                }
            default:
                continue
        }
        ctrl <- "next"
        //m6800.Step(mmu)
    }
    screen.Fini()
}

func box(s tcell.Screen, x, y, w, h int) {
    style := tcell.StyleDefault.Foreground(tcell.ColorGray)
    // corners
    s.SetContent(x, y, tcell.RuneULCorner, nil, style)
    s.SetContent(x+w, y, tcell.RuneURCorner, nil, style)
    s.SetContent(x, y+h, tcell.RuneLLCorner, nil, style)
    s.SetContent(x+w, y+h, tcell.RuneLRCorner, nil, style)
    // top/bottom
    for col := x+1; col < x+w; col++ {
        s.SetContent(col, y, tcell.RuneHLine, nil, style)
        s.SetContent(col, y+h, tcell.RuneHLine, nil, style)
    }
    // left/right
    for row := y+1; row < y+h; row++ {
        s.SetContent(x, row, tcell.RuneVLine, nil, style)
        s.SetContent(x+w, row, tcell.RuneVLine, nil, style)
    }
}

// draw 128 bytes from ram
func ramBox(s tcell.Screen, x, y int, label string, addr uint16, mem mem.MMU16) {
    box(s, x, y, 57, 11)
    style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Underline(true)
    colhead := "x0 x1 x2 x3 x4 x5 x6 x7  x8 x9 xA xB xC xD xE xF"
    drawString(s, x+8, y+2, style, colhead)
    style = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
    drawString(s, x+6, y, style, " "+label+" ")

    row := 0
    for {
        style = tcell.StyleDefault.Foreground(tcell.ColorWhite)
        low := int(addr&0xF)
        if low == 0 {
            rowhead := fmt.Sprintf("$%.4X", addr&0xFFF0)
            drawString(s, x+2, row+y+3, style, rowhead)
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
        drawString(s, col+4, row+y+3, style, cell)
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
    box(s, x, y, 20, 11)
    style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
    drawString(s, x+2, y, style, " M6802 ")
    style = tcell.StyleDefault.Foreground(tcell.ColorGray)
    col := x+2
    row := y+2
    drawString(s, col, row, style, "PC:")
    drawString(s, col, row+1, style, "SP:")
    drawString(s, col, row+2, style, " X:")
    drawString(s, col, row+3, style, " A:")
    drawString(s, col, row+4, style, " B:")
    drawString(s, col, row+6, style, "      --HINZVC")
    drawString(s, col, row+7, style, "CC:")
    style = tcell.StyleDefault.Foreground(tcell.ColorWhite)
    col = col + 4
    drawString(s, col, row, style, fmt.Sprintf("$%.4X ($%.4X)", cpu.PC&0x7ff, cpu.PC))
    drawString(s, col, row+1, style, fmt.Sprintf("$%.4X", cpu.SP))
    drawString(s, col, row+2, style, fmt.Sprintf("$%.4X", cpu.X))
    drawString(s, col, row+3, style, fmt.Sprintf("0x%.2X", cpu.A))
    drawString(s, col, row+4, style, fmt.Sprintf("0x%.2X", cpu.B))
    drawString(s, col, row+7, style, fmt.Sprintf("0b%.8b", cpu.CC))
}
