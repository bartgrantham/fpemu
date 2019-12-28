package main

import (
    "fmt"
    "io/ioutil"
//    "log"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/bartgrantham/fpemu/cpu/m6800"
    "github.com/bartgrantham/fpemu/mem"
//    "github.com/bartgrantham/fpemu/mem/firepower"
    "github.com/bartgrantham/fpemu/mem/d8224"
    "github.com/bartgrantham/fpemu/misc/hc55516"
    "github.com/bartgrantham/fpemu/pia/m6821"
    "github.com/bartgrantham/fpemu/ui"
    "github.com/gdamore/tcell"
)

// https://github.com/mamedev/mame/blob/master/src/mame/drivers/williams.cpp#L1899
// https://www.myplacearcade.com/wms_snd.php

/*
Sources of bugs:

* opcode logic
    * not advancing PC correctly for >1 byte opcodes
    * flags incorrectly set
    * cut and paste bugs
    * outright logic bugs
* opcode timing
* pia state machine
* other hardware features unaccounted for
* hc55516.CVSD bugs (unlikely)

*/

var presets map[string]string = map[string]string{
    "blaster"   : "F000=roms/blaster/blaster.18",
    "blackout"  : "B000=roms/blackout/V_IC7.532 C000=roms/blackout/V_IC5.532 D000=roms/blackout/V_IC6.532 F800=roms/blackout/SOUND2.716",
    "bubbles"   : "F000=roms/bubbles/bubbles.snd",
    "colony7"   : "F800=roms/colony7/cs11.bin",
    "defender"  : "F800=roms/defender/defend.snd",
    "firepower" : "B000=roms/firepower/V_IC7.532 C000=roms/firepower/V_IC5.532 D000=roms/firepower/V_IC6.532 F800=roms/firepower/SOUND3.716",
    "gorgar"    : "B000=roms/gorgar/v_ic7.532 C000=roms/gorgar/v_ic5.532 D000=roms/gorgar/v_ic6.532 F800=roms/gorgar/sound2.716",
    "inferno"   : "E000=roms/inferno/ic8.inf",
    "joust"     : "F000=roms/joust/joust.snd",
//    "joust2"    : "",
    "junglelord" : "B000=roms/junglelord/speech7.532 C000=roms/junglelord/speech5.532 D000=roms/junglelord/speech7.532 F800=roms/junglelord/sound3.716",
    "lasercue"  : "F800=roms/lasercue/sound12.716",
    "lottofun"  : "F000=roms/lottofun/vl2532.snd",
    "mayday"    : "F800=roms/mayday/ic28-8.bin",
    "mysticmarathon" : "E000=roms/mysticm/mm01_1.a08",
    "playball"  : "B000=roms/playball/speech.ic4 C000=roms/playball/speech.ic5 D000=roms/playball/speech.ic6 E000=roms/playball/speech.ic7 F000=roms/playball/playball.snd",
    "robotron2084" : "F000=roms/robotron2084/robotron.snd",
    "sinistar"  : "B000=roms/sinistar/speech.ic7 C000=roms/sinistar/speech.ic5 D000=roms/sinistar/speech.ic6 E000=roms/sinistar/speech.ic4 F000=roms/sinistar/sinistar.snd",
    "splat"     : "F000=roms/splat/splat.snd",
    "stargate"  : "F800=roms/stargate/sg.snd",
    "starlight" : "F800=roms/starlight/sound3.716",
    "timefantasy" : "F800=roms/timefantasy/sound3.716",
    "turkeyshoot" : "E000=roms/tshoot/rom1.cpu",
}

/*
    Awesome:
Blaster: 05, 0A, 19, 2B, 33/34/35/36/37
Bubbles: 0C, 0E, 10
Defender (pretty much all): 06, 07, 0A, 0B, 0D,
Firepower (pretty much all): 09, 0A, 2F, 31, 38, 
Joust (pretty much all): 07, 10, 13, 14/15,
Lasercue: 02, 04
Robotron 2084: 0A (like Defender), 27,
Sinistar: 06, 0B, 0D, 0E, 0F, 13, 19, 1B, 1D,
Splat: 0D, 12, 13/14, 16

    Crashes:
Blaster: 1C, 2C, 2E, 2F, 30, 31, 32, 
Colony7: crashes immediately (accesses $8401)
Inferno: crashes immediately (accesses $2001)
Joust: 24/25/26/27
Lotto Fun: almost nothing works, most stuff crashes, definitely goes off the rails with 18 -> invalid opcode 75
Mystic Marathon: crashes immediately (accesses $2001)
Playball: crashes on all sounds?
Sinistar: 03, 05
Splat: 07, 08
Turkey Shoot: crashes immediately (accesses $2001)

    Sounds wrong, or no sound:
Blaster: 0E, 0F, 13, 14, 16, 17, 18, 38, 3C, 3E, 3F
Bubbles: 0B (supposed to repeat?), 13, 14, 15, 16, 19, 1A, 1B
Defender: 13, 14, 16, 17, 18, 1A, 1B, 1C, 1D, 1E, 1F
Firepower: 12, 13, 14, 19, 1A, 1B, 1C, 1D, 1E, 1F, 39..3F
Joust: 18, 19, 1C?, 1F, 28..3F
Mayday: 0E, 0F, 13, 14, 16, 17, 18, 1C..1F
Robotron: 2D, 2E?, 35, 37, 3C, 3E, 3F
Sinistar: 01, 04, 05?, 07, 08, 0A, 1C?, 
Splat: 01, 02, 03, 05, 06, 18 (supposed to repeat?), 1D (supposed to repeat?)

    to guard against panics ruining your tty: export OLDSTTY=`stty -g`; go run emulator.go; stty $OLDSTTY;

    Pretty sure I need to do another opcode trace on some of the "too fast" ones (I, J, M) on firepower, they should
        be awesome phasing sounds instead of little blips

    Sounds not accurate:
    * 3 : probably perfect, might want to check again
    * f : missing "FIRE POWER MISSING ACCOMPLISHED", the sweep should be noise instead of tone
    * 12 : screeching instead of cool "engine" sound
    * 13 : silence (correct?)
    * 14 : should be ball launch sound (seems to be reading from ROM, sound file?)
    * 15 : missing "MISSION ONE", different sound (mine is "engine" sound, video is slow laser beam sweep)
    * 17 : sweep should be noise instead of tone?
    * 18 : "FIRE ONE" followed by BRRRR, I have a weird tone
    * 19 : definitely wrong, should be awesome phase bewwwwwww sound
    * 1a : "POWER", followed by another cool phasing bewwww sound
    * 1b : "FIRE DESTROYED YOU", followed by a really cool Robotron sound
    * 1c : AWESOME phasing sound
    * 1d : AWESOME phasing sound
    * 1e : "FIRE POWER", 1c in reverse?
    * 1f : "FIRE POWER", another awesome sweeping sound

TODO:

* breakpoints, "dirty" flags for what's changed between displays
* doing some disassemply to determine where the CVSD waveforms are, but this may not be clear until a LOT of it has been documented
* draw disasm
* load queue of commands from cli
* keys for: pause, step, 1x .1s .01s .001s
* draw PIA output
* keydown event loop for debugger, breakpoints on/off, continue, 1s/.1s/.01s/.001s/full speeds

*/

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: fpemu addr=roms/foo addr=roms/bar ...")
        os.Exit(-1)
    }

    // Init emulation
    ctrl := make(chan uint8, 10)
    cvsd := hc55516.CVSD{}
    pia := m6821.M6821{CVSD:cvsd}
    mmu := d8224.NewD8224Mem(&pia)
    var mountspecs []string
    if len(os.Args) == 2 && ! strings.ContainsAny(os.Args[1], "=") {
        preset, ok := presets[os.Args[1]]
        if !ok {
            fmt.Println("unknown preset", os.Args[1])
            fmt.Printf("Available presets: ")
            for k, _ := range presets {
                fmt.Printf("%s ", k)
            }
            fmt.Println()
            os.Exit(-1)
        }
        mountspecs = strings.Fields(preset)
    } else {
        mountspecs = os.Args[1:]
    }
    for _, arg := range mountspecs {
        parts := strings.Split(arg, "=")
        if len(parts) < 2 {
            fmt.Println("Invalid argument", arg)
            os.Exit(-1)
        }
        tmp, err := strconv.ParseInt(parts[0], 16, 32)
        if err != nil || tmp >= 1<<16 {
            fmt.Println("Invalid address", arg, `"`, err, `"`)
            os.Exit(-1)
        }
        addr := uint16(tmp)
        fh, err := os.Open(parts[1])
        if err != nil {
            fmt.Println("Can't open file", parts[1], ":", err)
            os.Exit(-1)
        }
        data, err := ioutil.ReadAll(fh)
        if err != nil {
            fmt.Println("Can't read file", parts[1], ":", err)
            os.Exit(-1)
        }

        fmt.Printf("mounting %s (%d bytes) at $%.4X\n", parts[1], len(data), addr)
        if err := mmu.Mount(addr, data); err != nil {
            fmt.Println("Can't mount:", err)
            os.Exit(-1)
        }
    }
    m6800 := m6800.NewM6800(mmu, &pia)

    // Init Host Audio
    err := ui.StartAudio(m6800.Callback(mmu, ctrl, &pia))
    if err != nil {
        fmt.Println("Couldn't start audio:", err)
        os.Exit(-1)
    }
    defer ui.StopAudio()

    // Init UI
    screen, err := tcell.NewScreen()
    if err != nil {
        fmt.Println("Error opening screen:", err)
        os.Exit(-1)
    }
    if err := screen.Init(); err != nil {
        fmt.Println("Error opening screen:", err)
        os.Exit(-1)
    }

    // Run UI
    bank := uint8(0)
    dl := []ui.Draw{func(){
        ramBox(screen, 3, 1, "IRAM", 0x0, mmu)
        cpuBox(screen, 64, 1, m6800, bank)
        ui.LogBox(screen, 3, 13, "Log")
    }}
    tui := ui.TextUI{
        Screen:screen,
        Tick:time.NewTicker(25 * time.Millisecond),
        DisplayList:dl,
    }
    tui.Run()

    defer func() {
        if r := recover(); r != nil {
        }
        screen.Fini()
        fmt.Println(m6800.Status())
        //ui.DumpLog()
    }()

    chr2code := map[rune]uint8{
        '1':0,  '2':1,  '3':2,  '4':3,  '5':4,  '6':5,  '7':6,  '8':7,
        'q':8,  'w':9,  'e':10, 'r':11, 't':12, 'y':13, 'u':14, 'i':15,
        'a':16, 's':17, 'd':18, 'f':19, 'g':20, 'h':21, 'j':22, 'k':23,
        'z':24, 'x':25, 'c':26, 'v':27, 'b':28, 'n':29, 'm':30, ',':31,
    }
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
                code, ok := chr2code[chr]
                switch {
                    case chr == '<' || chr == '>':
                        if chr == '<' {
                            bank -= 1
                        } else {
                            bank += 1
                        }
                        if bank < 0 {
                            bank = 8 - (-bank % 8)
                        } else {
                            bank = bank % 8
                        }
                    case ok:
                        ctrl <- bank*32 + code
                }
        }
    }
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


func cpuBox(s tcell.Screen, x, y int, cpu *m6800.M6800, bank uint8) {
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

    bankstr := fmt.Sprintf("BANK : %.2x", bank)
    ui.DrawString(s, col, row+8, style, bankstr)
}
