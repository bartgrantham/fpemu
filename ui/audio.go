package ui

//typedef unsigned char Uint8;
//void Callback(void *userdata, Uint8 *stream, int len);
import "C"
import (
    "log"
    "reflect"
    "unsafe"

    "github.com/veandco/go-sdl2/sdl"
)

/*
What is happening here?

    The emulator exports an audio callback that takes a []float32 and fills
it with samples.

    SDL requires that we export a C symbol `Callback` with the function
signature: void Callback(void *userdata, Uint8 *stream, int len);

    SDL is inflexible about this callback signature, and we don't know
ahead of time 1) the emulator's callback and 2) what sample size SDL will
be able to support (ie. requested vs. obtained AudioSpec).  We _also_
don't know the format until run-time, but we should expect
44.1KHz/16-bit/stereo to be widely supported.

    The solution is to declare a generator function, which gets set when
StartAudio() is called, and the generator's buffer, which gets allocated at
runtime (also in StartAudio()) based on the obtained callback buffer size.
Then when SDL calls `Callback` we can use (and re-use) our []float32 buffer,
and then copy it into the raw stream buffer SDL gives us.

    This allows us to bridge the variable callback buffer size, different
sample formats, and even do some DSP in the middle.

*/

var generator func([]float32)
var genbuf    []float32
var dev       sdl.AudioDeviceID

//export Callback
func Callback(userdata unsafe.Pointer, stream *C.Uint8, length C.int) {
    if generator != nil {
        generator(genbuf)
    }
    n := int(length) / 2
    hdr := reflect.SliceHeader{
        Data: uintptr(unsafe.Pointer(stream)),
        Len: n,
        Cap: n,
    }
    buf := *(*[]int16)(unsafe.Pointer(&hdr))
    for i:=0; i<n; i+=2 {
        if i/2 > len(genbuf) {
            break
        }
        buf[i] = int16(genbuf[i/2] * 2000)
        buf[i+1] = int16(genbuf[i/2] * 2000)
    }
}


func StartAudio(gen func([]float32)) error {
    var err error
    var count int

    if err = sdl.Init(sdl.INIT_AUDIO); err != nil {
        log.Println("SDL Audio Init error:", err)
        return err
    }

    count = sdl.GetNumAudioDrivers()
    log.Println("SDL Audio Drivers:")
    for i:=0; i<count; i++ {
        name := sdl.GetAudioDriver(i)
        log.Printf("    %d: %s\n", i, name)
    }

    count = sdl.GetNumAudioDevices(false)
    log.Printf("SDL Audio Devices:")
    for i:=0; i<count; i++ {
        name := sdl.GetAudioDeviceName(i, false)
        log.Printf("    %d: %s\n", i, name)
        // would like to print channels, sample rate, etc. but not available unless we init
    }

    generator = gen
    requested := sdl.AudioSpec{
        Freq:     44100,
        Format:   sdl.AUDIO_S16, // signed 16-bit floats
        Channels: 2,             // stereo
//        Samples:  256,           // 5.8ms at 44.1KHz
        Samples:  512,           // 11.6ms at 44.1KHz
//        Samples:  2048,           // 46.4ms at 44.1KHz
        Callback: sdl.AudioCallback(C.Callback),
    }
    obtained := sdl.AudioSpec{}

    log.Println("SDL Audio Spec Requested:", requested)
    if dev, err = sdl.OpenAudioDevice("", false, &requested, &obtained, 0); err != nil {
        log.Println("SDL OpenAudioDevice error:", err)
        return err
    }
    log.Println("SDL Audio Spec Obtained :", obtained)
    genbuf = make([]float32, obtained.Samples)

    sdl.PauseAudioDevice(dev, false)
    return nil
}

func StopAudio() {
    // anything else needed before quitting?
    sdl.AudioQuit()
}

