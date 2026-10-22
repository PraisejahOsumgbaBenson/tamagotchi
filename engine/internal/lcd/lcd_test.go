package lcd

import "testing"

func TestSetAndPixel(t *testing.T) {
	var s Screen

	s.Set(3, 4, true)
	if !s.Pixel(3, 4) {
		t.Error("the pixel at 3,4 should be on")
	}
	if s.Pixel(4, 3) {
		t.Error("the pixel at 4,3 should be off")
	}

	s.Set(3, 4, false)
	if s.Pixel(3, 4) {
		t.Error("the pixel at 3,4 should be off after being turned off")
	}
}

func TestSetIgnoresOutOfRangePixels(t *testing.T) {
	var s Screen

	s.Set(-1, 0, true)
	s.Set(0, -1, true)
	s.Set(Width, 0, true)
	s.Set(0, Height, true)

	if s.Icons() != 0 {
		t.Error("no icons should be lit")
	}
	for _, row := range s.Rows() {
		if row != 0 {
			t.Errorf("the screen should be empty, got a row of %b", row)
		}
	}
}

func TestRowsAreTopDown(t *testing.T) {
	var s Screen

	s.Set(0, 0, true)
	s.Set(31, 15, true)

	rows := s.Rows()
	if len(rows) != Height {
		t.Fatalf("got %d rows, want %d", len(rows), Height)
	}
	if rows[0]&1 == 0 {
		t.Error("row 0 should have the leftmost pixel on")
	}
	if rows[15]&(1<<31) == 0 {
		t.Error("row 15 should have the rightmost pixel on")
	}
}

func TestIcons(t *testing.T) {
	var s Screen

	s.SetIcon(IconAttention, true)
	if !s.Icon(IconAttention) {
		t.Error("the attention icon should be on")
	}
	if s.Icon(IconFeed) {
		t.Error("the feed icon should be off")
	}
	if s.Icons() != 1<<uint(IconAttention) {
		t.Errorf("icons = %b, want %b", s.Icons(), 1<<uint(IconAttention))
	}

	s.SetIcon(IconAttention, false)
	if s.Icons() != 0 {
		t.Errorf("icons = %b, want 0", s.Icons())
	}
}

func TestEqual(t *testing.T) {
	var a, b Screen

	a.Set(1, 1, true)
	if a.Equal(b) {
		t.Error("two different screens should not be equal")
	}

	b.Set(1, 1, true)
	if !a.Equal(b) {
		t.Error("two identical screens should be equal")
	}
}

func TestDrawSpriteClipsAtTheEdges(t *testing.T) {
	var s Screen

	sprite := Sprite{
		Width:  4,
		Height: 2,
		Rows:   []uint32{0b1111, 0b1111},
	}
	s.DrawSprite(Width-2, Height-1, sprite)

	if !s.Pixel(Width-2, Height-1) || !s.Pixel(Width-1, Height-1) {
		t.Error("the part of the sprite inside the screen should be drawn")
	}
	if s.Pixel(Width-2, Height-2) {
		t.Error("the part of the sprite above the screen should be clipped")
	}
}

func TestDrawTextFits(t *testing.T) {
	var s Screen

	end := s.DrawText(0, 0, "42")
	if end != TextWidth("42") {
		t.Errorf("DrawText returned %d, want %d", end, TextWidth("42"))
	}
	if !s.Pixel(0, 0) {
		t.Error("the first pixel of the first glyph should be on")
	}
}

func TestFillTextDrawsLettersInABar(t *testing.T) {
	const text = "88"

	var plain, bar Screen
	plain.DrawText(0, 0, text)
	bar.FillText(0, 0, text)

	if plain.Equal(bar) {
		t.Error("the bar text should differ from the plain text")
	}

	// Every pixel of the block is lit except the letters, which are holes.
	// The gap between the two glyphs is part of the block.
	for y := 0; y < glyphHeight; y++ {
		for x := 0; x < TextWidth(text); x++ {
			want := !plain.Pixel(x, y)
			if bar.Pixel(x, y) != want {
				t.Fatalf("pixel %d,%d = %v, want %v", x, y, bar.Pixel(x, y), want)
			}
		}
	}

	if bar.Pixel(TextWidth(text), 0) {
		t.Error("the block should stop at the end of the text")
	}
}

func TestLowercaseUsesTheUppercaseGlyph(t *testing.T) {
	var a, b Screen

	a.DrawText(0, 0, "abc")
	b.DrawText(0, 0, "ABC")
	if !a.Equal(b) {
		t.Error("lowercase should use the uppercase glyph")
	}
}
