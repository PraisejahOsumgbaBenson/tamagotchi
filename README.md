# tama

A photorealistic Tamagotchi P1 that floats on your macOS desktop.

The goal here is realism, and that word carries a lot of weight. Not "a virtual
pet that resembles a Tamagotchi" but the actual Tamagotchi: the original Bandai
ROM running on an emulated Epson E0C6S46 processor, driving a 32 by 16 pixel
monochrome LCD, the original three buttons, and the original piezo speaker.
The shell around it is drawn in CSS so that it looks like the plastic it is.

That tiny 32 by 16 screen is the whole trick. Render a pet on a big crisp
canvas and your brain says "simulation". Render it on 32 by 16 pixels with
visible gaps between the cells, an olive STN backlight and a glass reflection,
and your brain says "Tamagotchi" before it registers a single pixel.

## Running it

```bash
make deps
make build
make run
```

## How it fits together

Two processes, split along the line that matters.

**The engine** is Go, pure Go, no cgo. It owns the pet: the processor
emulation, the LCD state, the buttons, the clock, save states. It speaks JSON
over a WebSocket on localhost and nothing else. It also runs headless, which is
how the emulator gets tested.

**The shell** is Electron. It owns the pixels: the photoreal shell in CSS, the
32 by 16 canvas, the button press physics, the WebAudio piezo. It is a
transparent frameless window that floats above everything, joins every Space,
and passes clicks through the space around the device.

The split matters because the engine is portable and the shell is not. The same
engine that drives the desktop build can be compiled to WebAssembly for a
browser version, or to TinyGo for an ESP32 with a small OLED. Neither needs
the shell.

```
  macOS                    engine (Go)                    shell (Electron)
  +-------------+          +-------------+                +-------------+
  |  Desktop    |  <--->  |  Processor  |  <-- WebSocket |  Photoreal  |
  |  +------->  |  frames |  Pet state  |  --> frames    |  LCD canvas |
  |  | floating |         |  Save state |  <-- buttons   |  Piezo audio|
  |  | P1 window|         +-------------+                +-------------+
  |  +---------+                ^
  |       ^                     |
  |       +---------------------+
  |      launchd / launch agent
  +-----------------------------+
```

## Status

| Piece | State |
|---|---|
| Photoreal P1 shell in CSS | Working |
| 32 by 16 LCD canvas with pixel grid | Working |
| Three buttons with press physics | Working |
| Piezo audio through WebAudio | Working |
| Floating always-on-top window | Working |
| Engine protocol, save states, clock sync | Working |
| Pet behaviour engine | Working, not the real ROM |
| Epson E0C6S46 emulation with the original ROM | Next |

The behaviour engine is a stand in for the real thing. It speaks the same
protocol, so swapping in the genuine ROM changes what happens inside the
engine and nothing at all about the shell.

## Documentation

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full design
- [docs/adr/](docs/adr/) for the decisions and the reasoning behind them

## A note on the ROM

The Tamagotchi P1 ROM is Bandai's copyright. No emulator repository ships it,
this one included. It is gitignored, it stays on your machine, and the build
will refuse to run without it. See [docs/ROM.md](docs/ROM.md).
