# The ROM

The Tamagotchi P1 runs on an Epson E0C6S46 processor. Its behaviour comes
from a 32 KB ROM that Bandai owns.

No emulator project ships it. `jcrona/tamalib`, `GaryZ88/ArduinoGotchi`,
`RBEGamer/TamagotchiESP32` and `tonypan2/tamagotchi-p1-web` all leave it out
for the same reason, and this repository does the same.

## Getting it

The ROM was extracted from a high resolution photograph of the die. The
photographs are published by
[siliconpr0n.org](https://siliconpr0n.org/map/bandai/tamagotchi-v1/), and the
MAME `tama` set that the emulator projects refer to is derived from them.

Set the URL and run the script. It verifies the hash before accepting the
file, so a wrong download fails loudly instead of producing a pet that
misbehaves in ways that are hard to trace back here.

```bash
ROM_URL="https://example.invalid/path/to/tama.zip" make rom
```

The script accepts a zip or a bare binary, verifies the hash below, and
writes the file to `roms/rom.bin`. `roms/` is gitignored.

## The hash

```
MD5   3fce172403c27274e59723bbb9f6d8e9
SHA1  4b4979cf92dc9d2fb6d7295a38f209f3da144f72
```

The engine refuses to start without a file that matches. A wrong ROM produces
a screen full of nonsense and no error message, which is a miserable way to
lose an afternoon.

## Format

The engine expects the raw binary. The original library loads the program as
an array of 12 bit instructions, while the ROM file stores 16 bit big endian
words, so the loader unpacks 16 down to 12. Getting this wrong yields a
processor that executes noise, so the hash check above is the first line of
defence and the golden trace is the second.
