package pet

import (
	"time"

	"github.com/PraisejahOsumgbaBenson/tamagotchi/engine/internal/lcd"
)

// The hearts used on the meter screens. They are three by three because
// four of them have to fit across the fifteen pixels to the right of the
// character.
var (
	heartFull  = sprite("#.#", "###", ".#.")
	heartEmpty = sprite("#.#", "#.#", ".#.")
)

// Render draws the pet and reports which status icons are lit and which
// menu icon is highlighted.
//
// The eight status icons are wired to separate segments of the glass and are
// not part of the pixel matrix, so they are reported rather than drawn.
func (p *Pet) Render(s *lcd.Screen, now time.Time) (icons uint32, selected uint32) {
	s.Clear()

	if p.Calling {
		icons |= 1 << uint(lcd.IconAttention)
	}

	switch p.View {
	case ViewMain:
		p.renderMain(s, now)
	case ViewFeed:
		p.renderChoice(s, "MEAL", "SNACK")
	case ViewLight:
		p.renderChoice(s, "ON", "OFF")
	case ViewPlay:
		p.renderPlay(s, now)
	case ViewMedicine:
		p.renderMedicine(s)
	case ViewDuck:
		p.renderDuck(s)
	case ViewMeter:
		p.renderMeter(s, now)
	case ViewDiscipline:
		p.renderChoice(s, "SCOLD", "SPARE")
	case ViewClock:
		p.renderClock(s, now)
	}

	if p.Selected >= 0 && p.Selected < len(MenuIcons) {
		selected = 1 << uint(MenuIcons[p.Selected])
	}

	return icons, selected
}

// renderMain draws the character and whatever is happening around it.
func (p *Pet) renderMain(s *lcd.Screen, now time.Time) {
	if p.Stage == StageEgg {
		// A still egg reads as a screenshot. It leans over every few
		// seconds, which is the only thing an egg can do.
		s.DrawSprite(8+eggRock(now), 0, egg)
		return
	}

	s.DrawSprite(8+motionX(p.Mood, now), motionY(p.Mood, now), p.face(now))

	if p.Sick {
		s.DrawSprite(24, 0, skull)
	}

	if p.Asleep {
		s.DrawSprite(26, 1, letterZ)
		s.DrawSprite(29, 4, letterZ)
	}

	for i := 0; i < p.Poops && i < 2; i++ {
		s.DrawSprite(24, 8-i*7, poop)
	}
}

// eggRock returns how far the egg has leaned, in pixels.
//
// It rocks for a moment, then rests, so the egg looks alive without looking
// like it is being shaken.
func eggRock(now time.Time) int {
	const (
		period = 4 * time.Second
		rock   = 900 * time.Millisecond
	)

	phase := now.UnixMilli() % period.Milliseconds()
	if phase >= rock.Milliseconds() {
		return 0
	}

	third := rock.Milliseconds() / 3
	switch {
	case phase < third:
		return -1
	case phase < 2*third:
		return 1
	}
	return 0
}

// motionX and motionY return how far the character has moved from its
// resting spot.
//
// There are no shades of grey on the glass, so movement is the clearest
// signal the pet has. A refusal is a shake, a meal is a chew, a snack is a
// hop. Without this a fed pet and an ignored pet look the same.
func motionX(m Mood, now time.Time) int {
	ms := now.UnixMilli()
	switch m {
	case MoodEating:
		return []int{-1, 0, 1, 0}[int(ms/140)%4]
	case MoodSad:
		return []int{-1, 1}[int(ms/110)%2]
	case MoodSick:
		return []int{-1, 0, 1, 0}[int(ms/500)%4]
	case MoodHatching:
		return []int{-1, 1}[int(ms/90)%2]
	}
	return 0
}

func motionY(m Mood, now time.Time) int {
	if m != MoodHappy && m != MoodPlaying {
		return 0
	}
	if (now.UnixMilli()/220)%3 == 0 {
		return -1
	}
	return 0
}

// face picks the sprite for the pet's current mood and stage.
func (p *Pet) face(now time.Time) lcd.Sprite {
	if p.Stage == StageEgg {
		return egg
	}
	if p.Dead() {
		return babyDead
	}
	if p.Mood == MoodHatching {
		return eggHatch
	}

	base := babyIdle
	switch p.Stage {
	case StageTeen:
		base = teenIdle
	case StageAdult:
		base = adultIdle
	}

	switch p.Mood {
	case MoodHappy:
		return babyHappy
	case MoodSad:
		return babySad
	case MoodEating:
		// Chewing: the mouth opens and closes.
		if (now.UnixMilli()/180)%2 == 0 {
			return babyEat
		}
		return base
	case MoodPlaying:
		return base
	}

	if p.Sick {
		return babySick
	}
	if p.Asleep {
		return babySleep
	}

	// The pet blinks now and then.
	if now.UnixNano()/int64(200*time.Millisecond)%17 == 0 {
		return blink(base)
	}
	return base
}

// blink closes the pet's eyes.
func blink(sp lcd.Sprite) lcd.Sprite {
	out := lcd.Sprite{
		Width:  sp.Width,
		Height: sp.Height,
		Rows:   append([]uint32(nil), sp.Rows...),
	}
	for i := 7; i <= 8 && i < len(out.Rows); i++ {
		out.Rows[i] = rowMask(out.Width)
	}
	return out
}

// rowMask returns a row with every pixel on.
func rowMask(width int) uint32 {
	if width <= 0 || width > 32 {
		return 0
	}
	return 1<<uint(width) - 1
}

// renderChoice draws a two option menu. The chosen option is shown in
// reverse video, the way the original highlights it.
func (p *Pet) renderChoice(s *lcd.Screen, a, b string) {
	p.drawOption(s, a, 2, p.Choice == 0)
	p.drawOption(s, b, 9, p.Choice == 1)
}

// drawOption draws one option, inverted when it is the current one.
func (p *Pet) drawOption(s *lcd.Screen, text string, y int, inverted bool) {
	x := (lcd.Width - lcd.TextWidth(text)) / 2
	if inverted {
		s.FillText(x, y, text)
		return
	}
	s.DrawText(x, y, text)
}

// renderClock draws the time.
//
// The original shows the hours and the minutes as four digits across the
// middle of the display.
func (p *Pet) renderClock(s *lcd.Screen, now time.Time) {
	t := now.Add(p.ClockOffset)
	text := itoa2(t.Hour()) + ":" + itoa2(t.Minute())

	x := (lcd.Width - lcd.TextWidth(text)) / 2
	s.DrawText(x, 4, text)

	if p.SetMode {
		s.DrawText(1, 12, "SET")
	}
}

// itoa2 turns a number into a two digit string.
func itoa2(n int) string {
	if n < 0 {
		n = 0
	}
	if n > 99 {
		n = 99
	}
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

// renderPlay draws the mini game.
func (p *Pet) renderPlay(s *lcd.Screen, now time.Time) {
	s.DrawSprite(8, 0, p.face(now))

	// The two guesses the pet can make.
	s.DrawText(1, 6, "<")
	s.DrawText(29, 6, ">")
}

// renderMedicine draws the medicine screen. The pet sits on the left, as it
// does on the meter, so the label has the right hand side to itself.
func (p *Pet) renderMedicine(s *lcd.Screen) {
	s.DrawSprite(0, 0, babySick)
	s.DrawText(17, 5, "MED")
}

// renderDuck draws the cleaning screen.
func (p *Pet) renderDuck(s *lcd.Screen) {
	s.DrawSprite(0, 0, babyIdle)
	s.DrawText(17, 5, "DUCK")
}

// renderMeter draws one of the four status pages. The character sits on the
// left and the numbers on the right, which is how the original lays it out.
func (p *Pet) renderMeter(s *lcd.Screen, now time.Time) {
	s.DrawSprite(0, 0, p.face(now))

	switch p.MeterPage {
	case 0:
		s.DrawText(17, 0, "AGE")
		s.DrawText(17, 6, itoa(p.Age))
		s.DrawText(17, 11, itoa(p.Weight)+"OZ")
	case 1:
		s.DrawText(17, 0, "DIS")
		p.drawHearts(s, 17, 9, p.Discipline)
	case 2:
		s.DrawText(17, 0, "FOO")
		p.drawHearts(s, 17, 9, p.Hunger)
	case 3:
		s.DrawText(17, 0, "HAP")
		p.drawHearts(s, 17, 9, p.Happy)
	}
}

// drawHearts draws four hearts, the first n of them filled.
func (p *Pet) drawHearts(s *lcd.Screen, x, y, n int) {
	for i := 0; i < 4; i++ {
		if i < n {
			s.DrawSprite(x+i*4, y, heartFull)
		} else {
			s.DrawSprite(x+i*4, y, heartEmpty)
		}
	}
}

// itoa turns a small number into a string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "0"
	}
	var buf [4]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
