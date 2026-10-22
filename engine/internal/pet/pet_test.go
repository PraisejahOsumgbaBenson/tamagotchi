package pet

import (
	"testing"
	"time"

	"github.com/PraisejahOsumgbaBenson/tama/engine/internal/lcd"
)

var epoch = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

func TestMenuSelection(t *testing.T) {
	p := New(epoch)

	if p.Selected != -1 {
		t.Fatalf("nothing should be selected at the start, got %d", p.Selected)
	}

	p.Press(ButtonA, epoch)
	if p.Selected != 0 {
		t.Errorf("the first press should select the first icon, got %d", p.Selected)
	}

	p.Press(ButtonA, epoch)
	if p.Selected != 1 {
		t.Errorf("the second press should select the second icon, got %d", p.Selected)
	}

	p.Press(ButtonC, epoch)
	if p.Selected != -1 {
		t.Errorf("cancel should clear the selection, got %d", p.Selected)
	}
}

func TestSelectionCyclesThroughEveryMenuIcon(t *testing.T) {
	p := New(epoch)

	p.Press(ButtonA, epoch)
	for i := 1; i < len(MenuIcons); i++ {
		p.Press(ButtonA, epoch)
	}
	if p.Selected != len(MenuIcons)-1 {
		t.Fatalf("got %d, want %d", p.Selected, len(MenuIcons)-1)
	}

	p.Press(ButtonA, epoch)
	if p.Selected != 0 {
		t.Errorf("the selection should wrap around, got %d", p.Selected)
	}
}

func TestFeedingAMeal(t *testing.T) {
	p := New(epoch)
	p.Hunger = 2
	p.Weight = 30

	p.Press(ButtonA, epoch) // select feed
	p.Press(ButtonB, epoch) // open the feed menu
	if p.View != ViewFeed {
		t.Fatalf("view = %v, want the feed view", p.View)
	}

	p.Press(ButtonB, epoch) // confirm the meal

	if p.Hunger != 3 {
		t.Errorf("hunger = %d, want 3", p.Hunger)
	}
	if p.Weight != 31 {
		t.Errorf("weight = %d, want 31", p.Weight)
	}
	if p.View != ViewMain {
		t.Errorf("view = %v, want the main view", p.View)
	}
}

func TestAFullPetRefusesToEat(t *testing.T) {
	p := New(epoch)
	p.Hunger = 4
	p.Weight = 30

	p.Press(ButtonA, epoch)
	p.Press(ButtonB, epoch)
	p.Press(ButtonB, epoch)

	if p.Hunger != 4 {
		t.Errorf("hunger = %d, want 4", p.Hunger)
	}
	if p.Weight != 30 {
		t.Errorf("weight = %d, want 30", p.Weight)
	}
	if p.Mood != MoodSad {
		t.Errorf("mood = %v, want sad", p.Mood)
	}
}

func TestASnackFillsHappinessAndWeighsMore(t *testing.T) {
	p := New(epoch)
	p.Happy = 2
	p.Weight = 30

	p.Press(ButtonA, epoch)
	p.Press(ButtonB, epoch)
	p.Press(ButtonA, epoch) // choose the snack
	p.Press(ButtonB, epoch)

	if p.Happy != 3 {
		t.Errorf("happiness = %d, want 3", p.Happy)
	}
	if p.Weight != 32 {
		t.Errorf("weight = %d, want 32", p.Weight)
	}
	if p.Hunger != 4 {
		t.Errorf("hunger = %d, want 4, a snack should not fill the belly", p.Hunger)
	}
}

func TestTheDuckCleansUp(t *testing.T) {
	p := New(epoch)
	p.Poops = 3

	p.Press(ButtonA, epoch) // feed
	p.Press(ButtonA, epoch) // light
	p.Press(ButtonA, epoch) // play
	p.Press(ButtonA, epoch) // medicine
	p.Press(ButtonA, epoch) // duck
	p.Press(ButtonB, epoch) // open the duck
	p.Press(ButtonB, epoch) // clean

	if p.Poops != 0 {
		t.Errorf("poops = %d, want 0", p.Poops)
	}
}

func TestTheMeterCyclesThroughFourPages(t *testing.T) {
	p := New(epoch)

	meter := -1
	for i, ic := range MenuIcons {
		if ic == lcd.IconMeter {
			meter = i
		}
	}
	if meter < 0 {
		t.Fatal("the meter is not in the menu")
	}

	p.Press(ButtonA, epoch)
	for p.Selected != meter {
		p.Press(ButtonA, epoch)
	}
	p.Press(ButtonB, epoch) // open the meter

	if p.View != ViewMeter {
		t.Fatalf("view = %v, want the meter view", p.View)
	}

	for want := 1; want < 4; want++ {
		p.Press(ButtonA, epoch)
		if p.MeterPage != want {
			t.Errorf("page = %d, want %d", p.MeterPage, want)
		}
	}

	p.Press(ButtonA, epoch)
	if p.MeterPage != 0 {
		t.Errorf("the pages should wrap around, got %d", p.MeterPage)
	}
}

func TestHungerRunsOut(t *testing.T) {
	p := New(epoch)
	p.Hunger = 2

	p.Tick(epoch.Add(hungerInterval))
	if p.Hunger != 1 {
		t.Errorf("hunger = %d, want 1", p.Hunger)
	}

	p.Tick(epoch.Add(2 * hungerInterval))
	if p.Hunger != 0 {
		t.Errorf("hunger = %d, want 0", p.Hunger)
	}
}

func TestThePetCallsWhenItIsHungry(t *testing.T) {
	p := New(epoch)
	p.Hunger = 0

	p.Tick(epoch)
	if !p.Calling {
		t.Error("a pet with no food should be calling for attention")
	}
	if p.CallReason != ReasonHungry {
		t.Errorf("reason = %v, want hungry", p.CallReason)
	}
}

func TestTheCallStops(t *testing.T) {
	p := New(epoch)
	p.Hunger = 0
	p.Tick(epoch)

	p.Tick(epoch.Add(callDuration + time.Second))
	if p.Calling {
		t.Error("the call should have stopped")
	}
}

func TestThePetGrowsAndAges(t *testing.T) {
	p := New(epoch)

	p.Tick(epoch.Add(ageInterval))
	if p.Age != 1 {
		t.Errorf("age = %d, want 1", p.Age)
	}
	if p.Stage != StageBaby {
		t.Errorf("stage = %v, want baby", p.Stage)
	}

	p.Tick(epoch.Add(5 * ageInterval))
	if p.Age != 5 {
		t.Errorf("age = %d, want 5", p.Age)
	}
	if p.Stage != StageAdult {
		t.Errorf("stage = %v, want adult", p.Stage)
	}
}

func TestThePetSleepsAtNight(t *testing.T) {
	night := time.Date(2026, 10, 22, 23, 0, 0, 0, time.UTC)
	p := New(night)

	p.Tick(night)
	if !p.Asleep {
		t.Error("the pet should be asleep at 11 in the evening")
	}

	morning := time.Date(2026, 10, 23, 8, 0, 0, 0, time.UTC)
	p.Tick(morning)
	if p.Asleep {
		t.Error("the pet should be awake at 8 in the morning")
	}
	if p.Age != 1 {
		t.Errorf("age = %d, want 1, the pet counts a year when it wakes", p.Age)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	p := New(epoch)

	var a, b lcd.Screen
	p.Render(&a, epoch)
	p.Render(&b, epoch)

	if !a.Equal(b) {
		t.Error("two renders of the same state should match")
	}
	if len(a.Rows()) != lcd.Height {
		t.Errorf("the render has %d rows, want %d", len(a.Rows()), lcd.Height)
	}
}

func TestRenderLightsTheAttentionIcon(t *testing.T) {
	p := New(epoch)
	p.Hunger = 0
	p.Tick(epoch)

	var s lcd.Screen
	icons, _ := p.Render(&s, epoch)
	if icons&(1<<uint(lcd.IconAttention)) == 0 {
		t.Error("the attention icon should be lit")
	}
}

func TestRenderMarksTheSelectedMenuIcon(t *testing.T) {
	p := New(epoch)
	p.Press(ButtonA, epoch)

	var s lcd.Screen
	_, sel := p.Render(&s, epoch)
	if sel != 1<<uint(lcd.IconFeed) {
		t.Errorf("sel = %b, want the feed icon", sel)
	}
}

func TestHoldingAButtonDoesNotActTwice(t *testing.T) {
	p := New(epoch)

	p.SetButtons(true, false, false, epoch)
	p.SetButtons(true, false, false, epoch.Add(comboWindow))

	if p.Selected != 0 {
		t.Fatalf("the press should have selected the first icon, got %d", p.Selected)
	}

	// Still holding the button down.
	p.SetButtons(true, false, false, epoch.Add(2*comboWindow))
	if p.Selected != 0 {
		t.Errorf("holding a button should not act again, got %d", p.Selected)
	}
}

func TestAWithCMutesTheSound(t *testing.T) {
	p := New(epoch)

	p.SetButtons(true, false, false, epoch)
	p.SetButtons(true, false, true, epoch.Add(10*time.Millisecond))

	if p.Sound {
		t.Error("the sound should be muted")
	}
	if p.Selected != -1 {
		t.Errorf("nothing should be selected, got %d", p.Selected)
	}
}

func TestAQuickTapActsWithoutWaitingOutTheComboWindow(t *testing.T) {
	p := New(epoch)

	// Down and up inside one frame, which is what the engine hands over
	// when the two reports arrive between frames.
	p.SetButtons(true, false, false, epoch)
	p.SetButtons(false, false, false, epoch.Add(5*time.Millisecond))

	if p.Selected != 0 {
		t.Errorf("a tapped A should select the first icon, got %d", p.Selected)
	}
}

func TestAAndCInOneFrameMutes(t *testing.T) {
	p := New(epoch)

	// Both presses arrive in the same frame, so there is no window to wait
	// out.
	p.SetButtons(true, false, true, epoch)

	if p.Sound {
		t.Error("the sound should be muted")
	}
	if p.Selected != -1 {
		t.Errorf("the mute should not also open the menu, got %d", p.Selected)
	}
}

func TestTheClockScreenShowsTheTime(t *testing.T) {
	p := New(epoch)

	p.SetButtons(false, true, false, epoch)
	if p.View != ViewClock {
		t.Fatalf("view = %v, want the clock", p.View)
	}

	var s lcd.Screen
	p.Render(&s, epoch)

	// Noon on the clock.
	found := false
	for y := 0; y < lcd.Height && !found; y++ {
		for x := 0; x < lcd.Width; x++ {
			if s.Pixel(x, y) {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("the clock screen should show something")
	}
}

func TestTheClockCanBeSet(t *testing.T) {
	p := New(epoch)
	p.SetButtons(false, true, false, epoch) // the clock
	p.SetButtons(true, false, false, epoch.Add(time.Millisecond))
	p.SetButtons(true, false, true, epoch.Add(2*time.Millisecond))

	if !p.SetMode {
		t.Fatal("holding A and pressing C should start setting the time")
	}

	before := p.ClockOffset
	p.Press(ButtonA, epoch.Add(3*time.Millisecond))
	if p.ClockOffset != before+time.Hour {
		t.Errorf("the offset is %v, want it moved by an hour", p.ClockOffset)
	}
}
