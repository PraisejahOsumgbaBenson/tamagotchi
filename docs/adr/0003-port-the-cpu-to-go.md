# 0003. Port the processor core to Go rather than linking the C library

Status: accepted

## Context

The reference implementation, `jcrona/tamalib`, is a C library under GPLv2.
It emulates the Epson E0C6S46 and is already ported to desktop, STM32,
Arduino, ESP32, Pebble, Palm OS and the browser. It is well tested and it
works.

There are three ways to use it.

Link it. The engine is Go, so this means cgo. It works on macOS and on Linux,
but cgo does not exist for `GOOS=js`, so the browser version would need a
different code path. It also drags GPLv2 into the shipped application.

Compile it to WebAssembly with emscripten and load it in the renderer. That
works and it is how the existing browser emulator does it. But then the engine
is not Go, it is C in a sandbox, and the Electron shell would be talking to
something that is not the engine we wrote.

Port it to Go. Roughly 2000 lines of C, mostly a processor core and a
hardware model.

## Decision

Port the processor core to Go. Keep the C library as a git submodule, used
only by the reference test harness.

## Consequences

The engine stays pure Go, so it cross compiles to WebAssembly and to TinyGo
without change. The browser version and the ESP32 version fall out of the same
codebase.

The port has to be exact. The ROM is sensitive to timing and to the memory
map, so "looks about right" is not good enough. The golden trace makes it
verifiable: the reference harness runs the C implementation against a virtual
clock and writes every frame and every audio event to a file. The Go port
replays that trace and the frames have to match byte for byte. A port without
that test is how you end up with a pet that hums instead of singing.

Porting means writing the processor from its behaviour, using the published
disassembly of the P1 ROM as a reference and the C library as the oracle. It
is more work than linking. It is also the work worth doing, because it is the
part of this project that is actually interesting.

GPLv2 stays out of the shipped application. The submodule is a development
dependency and is not distributed.
