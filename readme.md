# Sunvox-go

[pkg.go.dev](https://pkg.go.dev/github.com/solarlune/sunvoxgo)

These are Go bindings made with [purego](https://github.com/ebitengine/purego) for [Sunvox](https://warmplace.ru/soft/sunvox/), the popular free modular tracker software.

Currently supported OSes _should_ be Windows (x86/x86-64), Linux (x86/x86-64/arm32/arm64), and Mac OS (x86-64/arm64). Only Linux has been tested overtly, with Windows being tested through Wine.

Apart from these bindings, the original developer library for Sunvox additionally has library builds for Javascript through WASM, Android, and iOS - testing these is currently outside of the scope of these bindings, but I'm not against adding support for more platforms, of course. The example for `sunvoxgo` packages the libraries for all supported platforms and architectures and removes the others.

## Installation

`go get github.com/solarlune/sunvoxgo`

## Usage

```go

// Get a reference to the engine; this is global to your app.
engine := sunvoxgo.Engine()

// Initialize the development library depending on target OS and arch using the library base directory.
absPath, err := filepath.Abs("./sunvox_lib-2.1.2b")
if err != nil {
    panic(err)
}

err = engine.InitFromDirectory(absPath, nil)

if err != nil {
    panic(err)
}

// Create an audio channel named "music".
channel, err := engine.CreateChannel("music")
if err != nil {
    panic(err)
}

// Load a Sunvox project and play it back.
channel.LoadFileFromFS(os.DirFS("./"), "assets/CityStatesOfGENOW.sunvox")

channel.PlayFromBeginning()

```

## Tips

- Channel.Seek() is slowest when executed on channels that are actively playing back music. It's faster on channels that aren't (so if you can rearrange the order of seeking and playing, that would be wise).
- Playback functions can be slow, so you can call them asynchronously if you don't need them to execute immediately.
- Sunvox can be very powerful; it includes the ability to read input from microphones and other audio input devices. This can cause hanging if used on a system with an audio server that only supports one application requesting the audio input at a time (i.e. Alsa on Linux), so it might be wise to remove that module if you don't expressly need it, or use an audio server that supports more options (Pulse, Jack, etc).

## Distribution

Build your app or game as usual, but include the relevant Sunvox development libraries / library directory (`sunvox_lib-2.1.2b` in the example) somewhere relative to your output executable so the libaries can be loaded dynamically at runtime.

## What's Implemented?

Most significantly-useful things that are available from the development library. There's still some areas that haven't been implemented, though, like adding and removing new modules or patterns, loading samples or instruments from files, or getting the audio scope / waveform for a module during playback.

Windows, Mac, and Linux support should work, but while the development library builds exist for mobile and web, I haven't implemented them. Web may be simple as the library is in a WASM format, so it might just need some glue code to call into Javascript to instantiate the WASM object and then tie the functions in Go to the functions implemented in the WASM (basically what purego already does for the Sunvox engine C libraries on desktop).

## LICENSE

The license for this Go bindings package itself is MIT. To use the bindings, however, you must adhere to the license outlined by the author of the development library (Nightradio), which can be found [here](example/sunvox_lib-2.1.2b/docs/license/LICENSE.txt).
