package ui

import (
    "fmt"
    "log"

    "github.com/gordonklaus/portaudio"
)

type Audio struct {
    stream    *portaudio.Stream
    fs        float64
    channels  int
    channel   chan float32
    ts        float64
}

var a *Audio

func NewAudio() *Audio {
    a := Audio{ts:0}
    a.channel = make(chan float32, 44100)
    return &a
}


func StartAudio(cb func([]float32)) error {
    portaudio.Initialize()

    log.Println(portaudio.VersionText())
    if ha, err := portaudio.HostApis(); err != nil {
        log.Fatal(err)
    } else {
        log.Println("Host APIs:")
        for _, hostapi := range ha {
            log.Println("   ", hostapi)
        }
    }

    if devices, err := portaudio.Devices(); err != nil {
        log.Fatal(err)
    } else {
        fmt.Println("Output devices:")
        for i, d := range devices {
            if d.MaxOutputChannels == 0 {
                continue
            }
            fmt.Printf("    %d) \"%s\" ; %d channels, %.1fkHz ; latency %s...%s\n",
                i, d.Name, d.MaxOutputChannels, d.DefaultSampleRate/1000, d.DefaultLowOutputLatency, d.DefaultHighOutputLatency)
        }
    }
    //fmt.Print(prompt + ": ")
    //if echo {
    //    fmt.Scan(&response)
    //
    do, _ := portaudio.DefaultOutputDevice()
    fmt.Println("Using output device:")
    fmt.Println("    OUT: ", do)

    a = NewAudio()

    host, err := portaudio.DefaultHostApi()
    if err != nil {
        log.Fatal(err)
    }

    parameters := portaudio.HighLatencyParameters(host.DefaultInputDevice, host.DefaultOutputDevice)
    a.fs = parameters.SampleRate
    a.channels = parameters.Output.Channels

    stream, err := portaudio.OpenStream(parameters,
        func (in, out []float32, _ portaudio.StreamCallbackTimeInfo, _ portaudio.StreamCallbackFlags){
            cb(out)
            return
        })
    if err != nil {
        return err
    }
    a.stream = stream

    if err := stream.Start(); err != nil {
        return err
    }
    return nil
}

func StopAudio() {
    a.stream.Close()
    portaudio.Terminate()
}
