# 0001. Split the engine from the shell

Status: accepted

## Context

The project has two kinds of work in it. One is the emulator: the Epson
E0C6S46 processor, the memory map, the timers, the LCD controller, the piezo
output. That work is exact, testable and worth getting right.

The other is presentation: the plastic shell, the pixel grid, the button
physics, the audio. That work is subjective, disposable, and nobody argues
about it in a code review.

Running them in one process mixes the two. Every emulator change would risk
breaking the visuals and every visual tweak would risk the emulator.

## Decision

Two processes.

The engine is Go. Pure Go, no cgo, so it can be cross compiled later without
change. It owns the pet, the screen state, the buttons, the clock and the save
file. It speaks JSON over a WebSocket on localhost. It also runs headless.

The shell is Electron. It owns the pixels and the sound. It is a transparent
frameless window that floats above everything.

## Consequences

The emulator can be tested without a display, and the visuals can be built
without an emulator. The protocol between them is small enough to read in a
terminal.

Electron is a heavy dependency for what is ultimately a small window. We accept
that because transparent frameless always-on-top windows are the one thing
Chromium gets exactly right on the first try, and the visuals are the point of
this project.

Two processes means two things to launch and one thing that can fail to start.
The shell must detect a missing engine and say so plainly rather than showing
an empty screen.

The engine must not assume the shell is there. It runs, it listens, and it
keeps the pet alive whether or not anyone is looking.
