// Package lcd models the Tamagotchi P1 display.
//
// The display is a 32 by 16 matrix of monochrome pixels. On top of that sits
// a set of eight status icons which are wired to separate segments of the
// glass, so they are not part of the matrix. They appear as four across the
// top of the glass and four across the bottom.
//
// The model here matches the original hardware exactly, which is the point:
// the shell draws whatever this says and nothing else, so the same engine
// can drive the desktop, the browser and a small OLED without changes.
package lcd

// The size of the pixel matrix.
const (
	Width  = 32
	Height = 16
)

// Icon identifies one of the eight status icons.
//
// The order is the order the original hardware wires them in: the first four
// sit across the top of the glass and the last four across the bottom. The
// last one, attention, is the only icon that is not a menu entry. It lights
// up on its own when the pet wants something.
type Icon uint8

// The status icons.
const (
	IconFeed Icon = iota
	IconLight
	IconPlay
	IconMedicine
	IconDuck
	IconMeter
	IconDiscipline
	IconAttention
	IconCount
)

// String names an icon.
func (i Icon) String() string {
	names := [...]string{
		"feed",
		"light",
		"play",
		"medicine",
		"duck",
		"meter",
		"discipline",
		"attention",
	}
	if i >= IconCount {
		return "unknown"
	}
	return names[i]
}

// Sprite is a bitmap with one bit per pixel. Pixels are packed into rows, and
// the leftmost pixel of a row is the least significant bit.
type Sprite struct {
	Width  int
	Height int
	Rows   []uint32
}

// Screen is the state of the display.
//
// The zero value is a valid screen with everything off.
type Screen struct {
	rows  [Height]uint32
	icons uint32
}

// Clear turns off every pixel and every icon.
func (s *Screen) Clear() {
	s.rows = [Height]uint32{}
	s.icons = 0
}

// Set turns a pixel on or off. Coordinates outside the matrix are ignored.
func (s *Screen) Set(x, y int, on bool) {
	if x < 0 || x >= Width || y < 0 || y >= Height {
		return
	}
	if on {
		s.rows[y] |= 1 << uint(x)
	} else {
		s.rows[y] &^= 1 << uint(x)
	}
}

// Pixel reports whether a pixel is on.
func (s *Screen) Pixel(x, y int) bool {
	if x < 0 || x >= Width || y < 0 || y >= Height {
		return false
	}
	return s.rows[y]&(1<<uint(x)) != 0
}

// SetIcon turns a status icon on or off.
func (s *Screen) SetIcon(i Icon, on bool) {
	if i >= IconCount {
		return
	}
	if on {
		s.icons |= 1 << uint(i)
	} else {
		s.icons &^= 1 << uint(i)
	}
}

// Icon reports whether a status icon is on.
func (s *Screen) Icon(i Icon) bool {
	return s.icons&(1<<uint(i)) != 0
}

// Icons returns the icon bitmask.
func (s *Screen) Icons() uint32 {
	return s.icons
}

// Rows returns a copy of the row bitmasks, bottom row first is not the case:
// row 0 is the top of the display.
func (s *Screen) Rows() []uint32 {
	out := make([]uint32, Height)
	copy(out, s.rows[:])
	return out
}

// Equal reports whether two screens show the same thing.
func (s Screen) Equal(other Screen) bool {
	return s.rows == other.rows && s.icons == other.icons
}

// DrawSprite draws a bitmap with its top left corner at x, y. Pixels that
// fall outside the matrix are clipped.
func (s *Screen) DrawSprite(x, y int, sp Sprite) {
	for row := 0; row < sp.Height && row < len(sp.Rows); row++ {
		bits := sp.Rows[row]
		for col := 0; col < sp.Width; col++ {
			if bits&(1<<uint(col)) == 0 {
				continue
			}
			s.Set(x+col, y+row, true)
		}
	}
}

// DrawHLine draws a horizontal run of pixels.
func (s *Screen) DrawHLine(x, y, length int) {
	for i := 0; i < length; i++ {
		s.Set(x+i, y, true)
	}
}

// DrawVLine draws a vertical run of pixels.
func (s *Screen) DrawVLine(x, y, length int) {
	for i := 0; i < length; i++ {
		s.Set(x, y+i, true)
	}
}

// DrawRect draws the outline of a rectangle.
func (s *Screen) DrawRect(x, y, w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	s.DrawHLine(x, y, w)
	s.DrawHLine(x, y+h-1, w)
	s.DrawVLine(x, y, h)
	s.DrawVLine(x+w-1, y, h)
}

// FillRect fills a rectangle.
func (s *Screen) FillRect(x, y, w, h int) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			s.Set(x+i, y+j, true)
		}
	}
}

// DrawTriangle draws a small upward pointing triangle, which is how the
// original marks the selected menu icon.
func (s *Screen) DrawTriangle(x, y int) {
	s.Set(x, y, true)
	s.Set(x-1, y+1, true)
	s.Set(x+1, y+1, true)
	s.Set(x-2, y+2, true)
	s.Set(x+2, y+2, true)
	s.Set(x-1, y+3, true)
	s.Set(x+1, y+3, true)
	s.Set(x, y+4, true)
}

// DrawText draws a string using the built in font and returns the x position
// just past the last glyph.
func (s *Screen) DrawText(x, y int, text string) int {
	if text == "" {
		return x
	}
	for _, r := range text {
		g, ok := font[normalise(r)]
		if !ok {
			g = font[' ']
		}
		for row := 0; row < glyphHeight; row++ {
			bits := g[row]
			for col := 0; col < glyphWidth; col++ {
				if bits&(1<<uint(col)) != 0 {
					s.Set(x+col, y+row, true)
				}
			}
		}
		x += glyphWidth + 1
	}
	return x - 1
}

// TextWidth returns the width in pixels a string will occupy.
func TextWidth(text string) int {
	if text == "" {
		return 0
	}
	return len(text)*(glyphWidth+1) - 1
}

// FillText draws text with the pixels inverted, which is how the original
// highlights the current choice in a menu.
func (s *Screen) FillText(x, y int, text string) {
	for _, r := range text {
		g, ok := font[normalise(r)]
		if !ok {
			g = font[' ']
		}
		for row := 0; row < glyphHeight; row++ {
			bits := g[row]
			for col := 0; col < glyphWidth; col++ {
				s.Set(x+col, y+row, bits&(1<<uint(col)) == 0)
			}
		}
		x += glyphWidth + 1
	}
}
