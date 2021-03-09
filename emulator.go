package main

import (
    "fmt"
    "io/ioutil"
    "os"
    "sort"
    "strconv"
    "strings"
    "time"

    "github.com/bartgrantham/fpemu/cpu/m6800"
    "github.com/bartgrantham/fpemu/mem"
    "github.com/bartgrantham/fpemu/mem/d8224"
    "github.com/bartgrantham/fpemu/misc/hc55516"
    "github.com/bartgrantham/fpemu/pia/m6821"
    "github.com/bartgrantham/fpemu/ui"
    "github.com/gdamore/tcell"
)

/*

Notes:

* this is the "GWave" sound engine, check AD's joust writeup for more info
* potential bugs
    * not advancing PC correctly for >1 byte opcodes
    * flags incorrectly set
    * cut and paste bugs
    * outright logic bugs
    * the C flag in add() still seems weird
    * the H flag in add() still seems weird
    * pia state machine
    * other hardware features unaccounted for
    * hc55516.CVSD bugs (its very simple, but maybe strange hardware abuse?)
    * double-checked: opcode timing (2021-03-06), flags (2021-03-08)
* documentation
    * http://www.8bit-era.cz/6800.html
    * https://github.com/mamedev/mame/blob/master/src/mame/drivers/williams.cpp#L1899
    *  https://www.myplacearcade.com/wms_snd.php
* x-platform windows build: `CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -tags static -ldflags "-s -w" emulator.go
* tty trick: `export OLDSTTY=$(stty -g); go run emulator.go; stty $OLDSTTY;`
* playball is a prototype so it may be correct that many of its sounds don't work, it may also have additional hardware I'm not emulating (was this the one with the AY chip?)
    * [video of the prototype](https://www.youtube.com/watch?v=hkhxUlyetBs) shows sounds I don't have
* Awesome:
    * Blaster       : 05, 0A, 11, 19, 1C!, 2B, 33/34/35/36/37 (has a lot of great noise sounds)
    * Blackout      : Voice parts 38-3F, tones 42-5F
    * Bubbles       : 0C, 0E, 10
    * Defender      : 06, 07, 0A!!, 0B, 0D, ...pretty much all
    * Firepower     : 09, 0A, 2F, 31, 38, ...pretty much all
    * Gorgar        : 12 (heartbeat!), 38..3f
    * Joust         : 07, 10, 13, 14/15, ...pretty much all
    * Lasercue      : 02, 04, 09, 0A (tons of great "electronica" sounds)
    * Pharaoh       : 0B!, 12, 14!, 31! (Lots of fun stuff onthe second page!)
    * Robotron 2084 : 0A (like Defender), 27,
    * Sinistar      : 06, 07!!, 0B, 0D, 0E, 0F, 13, 19, 1B, 1D,
    * Splat         : 0D, 12, 13/14, 16 (lots of little melodies in 00-07)
    * Thunderball   : 01, 16 (waves+birds!), (has a lot of synthy "breakdown" sounds)
* Crashes:
    * Colony7         : crashes immediately on reading $8401
    * Inferno         : crashes immediately on write to $2001
    * Lotto Fun       : almost nothing works, most stuff crashes on reading from $BF5F
    * Mystic Marathon : crashes immediately on write to $2001
    * Playball        : needs RTI implemented
    * Turkey Shoot    : crashes immediately on write to $2001
* Sounds wrong, or no sound
    * Blaster      : 18?, 26, 3F?
    * Blackout     : 08?
    * Bubbles      : 0B (supposed to repeat?), 12 (supposed to have rumble afterwards?)
    * Defender     : 1B, 1C
    * Gorgar       : 08
    * Joust        : 1C?, 1E, 1F, 28..3F?
    * Mayday       : 1B, 1C
    * Pharaoh      : 11, 15
    * Playball     : almost all silent on bank 1, half of bank 2 silent, many sounds are small blips
    * Robotron     : 1B, 1C
    * Sinistar     : 01, 04, 08, 0A, 14, 1C?, 1E, 1F
    * Stargate     : 1B, 1C
    * Thunderball  : 3A-3F

TODO:

* save file to output.wav
* breakpoints, "dirty" flags for what's changed between displays
* doing some disassemply to determine where the CVSD waveforms are, but this may not be clear until a LOT of it has been documented
* load queue of commands from cli
* draw PIA output
* keydown event loop for debugger, breakpoints on/off, continue, 1s/.1s/.01s/.001s/full speeds

*/

var presets map[string]string = map[string]string{
    "blaster"   : "F000=roms/blaster/blaster.18",
    "blackout"  : "B000=roms/blackout/V_IC7.532 C000=roms/blackout/V_IC5.532 D000=roms/blackout/V_IC6.532 F800=roms/blackout/SOUND2.716",
    "bubbles"   : "F000=roms/bubbles/bubbles.snd RAM=EFFD,DFFD",
    "colony7"   : "F800=roms/colony7/cs11.bin",
    "defender"  : "F800=roms/defender/defend.snd RAM=EFFD",
    "firepower" : "B000=roms/firepower/V_IC7.532 C000=roms/firepower/V_IC5.532 D000=roms/firepower/V_IC6.532 F800=roms/firepower/SOUND3.716",
    "gorgar"    : "B000=roms/gorgar/v_ic7.532 C000=roms/gorgar/v_ic5.532 D000=roms/gorgar/v_ic6.532 F800=roms/gorgar/sound2.716",
    "inferno"   : "E000=roms/inferno/ic8.inf",
    "joust"     : "F000=roms/joust/joust.snd",
    "junglelord" : "B000=roms/junglelord/speech7.532 C000=roms/junglelord/speech5.532 D000=roms/junglelord/speech6.532 F800=roms/junglelord/sound3.716",
    "lasercue"  : "F800=roms/lasercue/sound12.716",
    "lottofun"  : "F000=roms/lottofun/vl2532.snd",
    "mayday"    : "F800=roms/mayday/ic28-8.bin RAM=EFFD",
    "mysticmarathon" : "E000=roms/mysticm/mm01_1.a08",
    "pharaoh"   : "B000=roms/pharaoh/speech7.532 C000=roms/pharaoh/speech5.532 D000=roms/pharaoh/speech6.532 E000=roms/pharaoh/speech4.532 F800=roms/pharaoh/sound12.716",
    "playball"  : "B000=roms/playball/speech.ic4 C000=roms/playball/speech.ic5 D000=roms/playball/speech.ic6 E000=roms/playball/speech.ic7 F000=roms/playball/playball.snd",
    "robotron2084" : "F000=roms/robotron2084/robotron.snd",
    "sinistar"  : "B000=roms/sinistar/speech.ic7 C000=roms/sinistar/speech.ic5 D000=roms/sinistar/speech.ic6 E000=roms/sinistar/speech.ic4 F000=roms/sinistar/sinistar.snd",
    "splat"     : "F000=roms/splat/splat.snd",
    "stargate"  : "F800=roms/stargate/sg.snd",
    "starlight" : "F800=roms/starlight/sound3.716 RAM=DFFD",
    "thunderball" : "B000=roms/thunderball/speech7.532 C000=roms/thunderball/speech5.532 D000=roms/thunderball/speech6.532 E000=roms/thunderball/speech4.532 F000=roms/thunderball/sound12.532",
    "timefantasy" : "F800=roms/timefantasy/sound3.716 RAM=DFFD",
    "turkeyshoot" : "E000=roms/tshoot/rom1.cpu",
}

func main() {
    var mountspecs []string
    var preset string
    var disasm bool

    for i, arg := range os.Args {
        if i == 0 {
            continue
        }
        switch {
            case arg == "--disasm":
                disasm = true
            case strings.IndexByte(arg, '=') > -1:
                mountspecs = append(mountspecs, arg)
            default:
                if tmp, ok := presets[arg]; ok {
                    preset = tmp
                } else {
                    fmt.Println("Unknown preset:", os.Args[1])
                    fmt.Println("Available presets:")
                    var names []string
                    for k, _ := range presets {
                        names = append(names, k)
                    }
                    sort.Strings(names)
                    out := ""
                    for _, name := range names {
                        if out == "" {
                            out += "    " + name
                        } else {
                            out += "  " + name
                        }
                        if len(out) > 70 {
                            fmt.Println(out)
                            out = ""
                        }
                    }
                    fmt.Println(out)
                    os.Exit(-1)
                }
        }
    }
    if preset != "" {
        mountspecs = strings.Fields(preset)
    }

    // no preset, no mountspecs: try to find a "sound.rom", case-insensitive, in cwd
    if len(mountspecs) == 0 {
        path, err := os.Getwd()
        if err != nil {
            fmt.Println(err)
            os.Exit(-1)
        }
        wd, err := os.Open(path)
        if err != nil {
            fmt.Println(err)
            os.Exit(-1)
        }
        files, err := wd.Readdir(-1)
        wd.Close()
        if err != nil {
            fmt.Println(err)
            os.Exit(-1)
        }
        for _, file := range files {
            if strings.ToLower(file.Name()) == "sound.rom" {
                mountspecs = append(mountspecs, "f000=" + file.Name())
                fmt.Println("found", file.Name())
                break
            }
        }
    }

    if len(mountspecs) == 0 {
        fmt.Println("Usage: fpemu addr=roms/foo addr=roms/bar ...")
        fmt.Println("   or: fpemu <romset>")
        fmt.Println("   or: fpemu   # must have \"sound.rom\" in the current directory\n")
        carriage := 0
        fmt.Printf("romsets: ")
        for name, _ := range presets {
            if carriage + len(name) > 70 {
                fmt.Printf("\n         ")
                carriage = 0
            }
            fmt.Printf(" %s", name)
            carriage += len(name)
        }
        fmt.Println("\n")
        os.Exit(-1)
    }

    // Init emulation
    ctrl := make(chan uint8, 10)
    cvsd := hc55516.CVSD{}
    pia := &m6821.M6821{CVSD:cvsd}
    mmu := d8224.NewD8224Mem(pia)
    for _, arg := range mountspecs {
        parts := strings.Split(arg, "=")
        if len(parts) < 2 {
            fmt.Println("Invalid argument", arg)
            os.Exit(-1)
        }
        if parts[0] == "RAM" {
            var start, end int64
            var err error
            addrs := strings.Split(parts[1], ",")
            for _, addr := range addrs {
                startend := strings.Split(addr, "-")
                if start, err = strconv.ParseInt(startend[0], 16, 32); err != nil {
                    fmt.Println("Invalid address", startend[0], `"`, err, `"`)
                    os.Exit(-1)
                }
                if len(startend) > 1 {
                    if end, err = strconv.ParseInt(startend[1], 16, 32); err != nil {
                        fmt.Println("Invalid address", startend[1], `"`, err, `"`)
                        os.Exit(-1)
                    }
                } else {
                    end = start + 1
                }
                if (start < 128) || (end >= 1<<16) || (start > end) {
                    fmt.Println("Invalid addresses:", start, end)
                        os.Exit(-1)
                }
                ram := make([]uint8, end-start)
                fmt.Printf("mounting %d bytes of RAM at $%.4X\n", len(ram), start)
                if err := mmu.Mount(uint16(start), ram, true); err != nil {
                    fmt.Println("Can't mount:", err)
                    os.Exit(-1)
                }
            }
            continue
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
        if err := mmu.Mount(addr, data, false); err != nil {
            fmt.Println("Can't mount:", err)
            os.Exit(-1)
        }
    }

    M6800 := m6800.NewM6800(mmu, pia)

    // Short-circuit for disasm
    if disasm {
        for addr:=0; addr<(1<<16); {
            out, adv := M6800.Disasm(uint16(addr), mmu)
            if adv != 0 {
                fmt.Println(out)
                addr += int(adv)
            } else {
                addr += 1
            }
        }
        os.Exit(0)
    }

    // Init Host Audio
    err := ui.StartAudio(M6800.Callback(mmu, ctrl, pia))
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
    m6800.Scr = screen

    // Run UI
    var last_chr rune
    var last_time time.Time
    bank := uint8(0)
    dl := []ui.Draw{func(){
        ramBox(screen, 3, 0, "IRAM", 0x0, mmu)
        cpuBox(screen, 64, 0, M6800, bank)
        //ui.LogBox(screen, 3, 13, "Log")
        kbBox(screen, 7, 12, bank, last_chr, last_time)
        quitBox(screen, 27, 23)
    }}
    tui := ui.TextUI{
        Screen:screen,
        Tick:time.NewTicker(50 * time.Millisecond),
        DisplayList:dl,
    }
    tui.Run()

    defer func() {
        if r := recover(); r != nil {
        }
        screen.Fini()
        fmt.Println(M6800.Status())
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
                    case ok:
                        ctrl <- bank*32 + code
                        last_chr = chr
                        last_time = time.Now()
                    case chr == '<' || chr == '>':
                        last_chr = chr
                        last_time = time.Now()
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

    vals, reads, writes := mem.Heat(0, 128)
    for addr, val := range vals {
        low := addr&0xF
        row := addr >> 4
        if low == 0 {
            style = tcell.StyleDefault.Foreground(tcell.ColorWhite)
            rowhead := fmt.Sprintf("$%.4X", addr&0xFFF0)
            ui.DrawString(s, x+2, row+y+3, style, rowhead)
        }
        col := x + (low*3) + 4
        if low >= 8 {
            col += 1
        }

        switch {
            case reads[addr] > 0 && writes[addr] > 0:
                style = tcell.StyleDefault.Foreground(tcell.ColorYellow)
            case writes[addr] > 0:
                style = tcell.StyleDefault.Foreground(tcell.ColorRed)
            case reads[addr] > 0:
                style = tcell.StyleDefault.Foreground(tcell.ColorGreen)
            default:
                style = tcell.StyleDefault.Foreground(tcell.ColorGray)
        }
        cell := fmt.Sprintf("%.2X", val)
        ui.DrawString(s, col+4, row+y+3, style, cell)
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

var bignums [][]string = [][]string{
    []string{
        "  d88  ",
        "   88  ",
        "   88  ",
        "   88  ",
        "   88  ",
        "  d88P ",
    },[]string{
        "d8888b.",
        "    `88",
        ".aaadP'",
        "88'    ",
        "88.    ",
        "Y88888P",
    },[]string{
        "d8888b.",
        "    `88",
        " aaad8'",
        "    `88",
        "    .88",
        "d88888P",
    },[]string{
        "dP   dP",
        "88   88",
        "88aaa88",
        "     88",
        "     88",
        "     dP",
    },
}

var kb_outline []string = []string{
    "          .---.---.---.---.---.---.---.---.",
    " $00..$07 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 |",
    "          '-.-'-.-'-.-'-.-'-.-'-.-'-.-'-.-'-.",
    " $08..$0f   | q | w | e | r | t | y | u | i |",
    "            '.--'.--'.--'.--'.--'.--'.--'.--'.",
    " $10..$17    | a | s | d | f | g | h | j | k |",
    "             '.--'.--'.--'.--'.--'.--'.--'.--'.    .---.        .---.",
    " $18..$1f     | z | x | c | v | b | n | m | , |    | < |  BANK  | > |",
    "              '---'---'---'---'---'---'---'---'    '---'        '---'",
}
var chr2x = map[rune]int{
    '1':12, '2':16, '3':20, '4':24, '5':28, '6':32, '7':36, '8':40,
    'q':14, 'w':18, 'e':22, 'r':26, 't':30, 'y':34, 'u':38, 'i':42,
    'a':15, 's':19, 'd':23, 'f':27, 'g':31, 'h':35, 'j':39, 'k':43,
    'z':16, 'x':20, 'c':24, 'v':28, 'b':32, 'n':36, 'm':40, ',':44,
    '<':53, '>':66,
}
var chr2y = map[rune]int{
    '1':1, '2':1, '3':1, '4':1, '5':1, '6':1, '7':1, '8':1,
    'q':3, 'w':3, 'e':3, 'r':3, 't':3, 'y':3, 'u':3, 'i':3,
    'a':5, 's':5, 'd':5, 'f':5, 'g':5, 'h':5, 'j':5, 'k':5,
    'z':7, 'x':7, 'c':7, 'v':7, 'b':7, 'n':7, 'm':7, ',':7,
    '<':7, '>':7,
}
var chr2time = map[rune]time.Time{}
func kbBox(s tcell.Screen, x, y int, bank uint8, last_chr rune, last_time time.Time) {
    now := time.Now()
    if last_time.Add(100 * time.Millisecond).After(now) {
        chr2time[last_chr] = now
    }

    ui.Box(s, x, y, 71, 10)
    letter := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
    key    := tcell.StyleDefault.Foreground(tcell.ColorGray)

    dx := x + 1
    dy := y + 1
    for j, str := range kb_outline {
        for i, c := range str {
            if c == '\'' || c == '-' || c == '.' || c == '|' {
                s.SetContent(dx+i, dy+j, c, []rune{}, key)
            } else {
                s.SetContent(dx+i, dy+j, c, []rune{}, letter)
            }
        }
    }

    var hl_x, hl_y int
    flash  := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true).Reverse(true)
    for chr, t := range chr2time {
        if t.Add(300 * time.Millisecond).Before(now) {
            delete(chr2time, chr)
            continue
        }
        hl_x, _ = chr2x[chr]
        hl_y, _ = chr2y[last_chr]
        s.SetContent(dx+hl_x, dy+hl_y, chr, []rune{}, flash)
    }
    dx = x + 57
    dy = y + 1
    for j, str := range bignums[bank%4] {
        for i, c := range str {
            s.SetContent(dx+i, dy+j, c, []rune{}, letter)
        }
    }

    var hexrange string
    var rangestart int
    for j:=0; j<4; j++ {
        rangestart = int(bank%4)<<5 + (j*8)
        hexrange = fmt.Sprintf("$%.2x..$%.2x", rangestart, rangestart+7)
        for i, c := range hexrange {
            s.SetContent(x+2+i, y+2+(j*2), c, []rune{}, letter)
        }
    }
}

func quitBox(s tcell.Screen, x, y int) {
    style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
    for i, c := range "---=== CTRL-C to quit ===---" {
        s.SetContent(x+i, y, c, []rune{}, style)
    }
}
