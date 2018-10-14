package ui

import (
    "fmt"
    "time"

    "github.com/gdamore/tcell"
)

var circlog []string //= make([]string, 1000)
var circlogidx int

func init() {
    circlog = make([]string, 1000)
}

func Log(msg string) {
    circlog[circlogidx] = msg
    circlogidx = (circlogidx+1) % 1000
}

func DumpLog() {
    for i, _ := range circlog {
        msg := circlog[999-i]
        if msg != "" {
            fmt.Println(i, msg)
        }
    }
}

func LogBox(s tcell.Screen, x, y int, label string) {
    Box(s, x, y, 100, 15)
    Clear(s, x+1, y+1, 98, 14)
    style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
    DrawString(s, x+2, y, style, " "+label+" ")
    for i:=0; i<12; i++ {
        li := circlogidx-12+i
        if li < 0 {
            li += 1000
        }
        DrawString(s, x+2, y+2+i, style, fmt.Sprintf("%d %s", li, circlog[li]))
    }
}

func DrawString(s tcell.Screen, x, y int, style tcell.Style, str string) {
    for _, c := range str {
        s.SetContent(x, y, c, []rune{}, style)
        x += 1
    }
}

func Box(s tcell.Screen, x, y, w, h int) {
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

func Clear(s tcell.Screen, x, y, w, h int) {
    style := tcell.StyleDefault//.Foreground(tcell.ColorGray)
    for col := x; col <= x+w; col++ {
        for row := y; row <= y+h; row++ {
            s.SetContent(col, row, ' ', nil, style)
        }
    }
}

type Draw func()

type TextUI struct {
    Screen  tcell.Screen
    Tick    *time.Ticker
    DisplayList  []Draw
}

func (t *TextUI) Run() {
    go func() {
        for {
            <-t.Tick.C
            for _, drawfunc := range t.DisplayList {
                drawfunc()
            }
            t.Screen.Show()
        }
    }()
}

