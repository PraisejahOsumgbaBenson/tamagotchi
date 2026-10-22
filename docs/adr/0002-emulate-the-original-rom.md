# 0002. Emulate the original ROM rather than imitate the pet

Status: accepted

## Context

There are two ways to build a Tamagotchi that behaves like a Tamagotchi.

The first is to study what the original does and reimplement it. Feed it
every day, play with it, clean it, let it get sick, let it die. Write a state
machine that reproduces the observed behaviour.

The second is to run the original code. The P1's behaviour is not a design
anyone wrote down, it is what happens when a particular 32 KB of ROM meets a
particular processor. It has surprises. It has bugs that became features. It
has specific behaviours that nobody would design and everybody remembers.

## Decision

Emulate the Epson E0C6S46 processor and run the original Bandai P1 ROM in it.

The LCD state, the button presses and the piezo output all come from the
emulated hardware rather than from a Go implementation of pet logic.

## Consequences

The behaviour is right by construction, not by approximation. Whatever the
original does, including the parts nobody documented, is what happens.

The project depends on a ROM that is Bandai's copyright. No emulator
repository ships it and neither do we. It is gitignored, it lives on the
machine, and the build checks for it and refuses to run without it. See
[../ROM.md](../ROM.md).

It also depends on the emulator matching the hardware closely enough that the
ROM behaves. That is not something to eyeball, so the C reference
implementation produces a golden trace of frames and audio events, and the Go
port has to reproduce it byte for byte.

Until the emulator is finished, a stand in pet engine speaks the same
protocol. That keeps the shell developable and the whole app runnable, and
it costs nothing later because swapping in the real ROM changes what happens
inside the engine and nothing about the shell.
