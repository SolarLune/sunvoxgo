# Sunvox-go

[pkg.go.dev](https://pkg.go.dev/github.com/solarlune/sunvoxgo)

These are Go bindings made with [purego](https://github.com/ebitengine/purego) for [Sunvox](https://warmplace.ru/soft/sunvox/), the popular free modular tracker software.

Currently supported OSes _should_ be Windows (x86/x86-64), Linux (x86/x86-64/arm32/arm64), and Mac OS (x86-64/arm64).

Apart from these bindings, the original developer library for Sunvox additionally support Javascript (through a WASM build), Android, and iOS - testing these is currently outside of the scope of these bindings, but I'm not against adding support for more platforms, of course. The example packages the libraries for all supported platforms and architectures and removes the others.

## LICENSE

The license for this Go bindings package itself is MIT. To use the bindings, however, you must adhere to the license outlined by the author of the development library (Nightradio), which can be found [here](example/sunvox_lib-2.1.2b/docs/license/LICENSE.txt).
