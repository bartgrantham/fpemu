package main

import (
    "fmt"
    "io/ioutil"
    "os"
    "time"

    "github.com/bartgrantham/fpemu/cpu/m6800"
    "github.com/bartgrantham/fpemu/mem/firepower"
)

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

    mmu := firepower.NewFirepowerMem(buf)
    m6800 := m6800.NewM6800(mmu)
    for i:=0; i<32; i++ {
        fmt.Println(firepower.Dump(mmu.IRAM))
        fmt.Println()
        fmt.Println(firepower.Dump(mmu.ORAM))
        fmt.Println()
        fmt.Println(m6800.Status())
        fmt.Println("-=-=-=-=-=-")
        m6800.Step(mmu)
        time.Sleep(1 * time.Second)
    }
}
