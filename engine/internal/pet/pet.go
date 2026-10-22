// Package pet holds the behaviour of the Tamagotchi.
//
// This is a stand in for the real thing. The genuine behaviour comes from the
// original ROM running on an emulated processor, and it will replace this
// package without changing anything on the wire. What lives here is the
// original rules, written out: four hearts of hunger, four hearts of
// happiness, meals that fill one heart and snacks that fill the other, a
// duck that cleans up after you, and a pet that dies if you ignore it.
package pet

import (
	"time"

	"github.com/PraisejahOsumgbaBenson/tama/engine/internal/lcd"
)

// Stage is a stage of life.
type Stage int

// The stages.
const (
	StageEgg Stage = iota
	StageBaby
	StageChild
	StageTeen
	StageAdult
)

// String names a stage.
func (s Stage) String() string {
	switch s {
	case StageEgg:
		return "egg"
	case StageBaby:
		return "baby"
	case StageChild:
		return "child"
	case StageTeen:
		return "teen"
	case StageAdult:
		return "adult"
	}
	return "unknown"
}

// View is what the pet is showing.
type View int

// The views.
const (
	ViewMain View = iota
	ViewFeed
	ViewLight
	ViewPlay
	ViewMedicine
	ViewDuck
	ViewMeter
	ViewDiscipline
	ViewClock
)

// Button is one of the three buttons on the device.
type Button int

// The buttons.
const (
	ButtonA Button = iota
	ButtonB
	ButtonC
)

// Reason says why the pet is calling for attention.
type Reason int

// The reasons.
const (
	ReasonNone Reason = iota
	ReasonHungry
	ReasonUnhappy
	ReasonLight
	ReasonNaughty
)

// MenuIcons are the icons that can be selected, in the order the original
// cycles through them. The attention icon is not in the list because it is an
// indicator rather than a menu entry.
var MenuIcons = []lcd.Icon{
	lcd.IconFeed,
	lcd.IconLight,
	lcd.IconPlay,
	lcd.IconMedicine,
	lcd.IconDuck,
	lcd.IconMeter,
	lcd.IconDiscipline,
}

// The tuning of the pet.
//
// The original device spreads all of this over hours, which suits a keychain
// and does not suit a pet that lives on a desktop. The clock here runs fast.
// The rules are the original rules.
const (
	hungerInterval = 8 * time.Minute
	happyInterval  = 12 * time.Minute
	ageInterval    = 15 * time.Minute

	poopIntervalBaby = 20 * time.Minute
	poopIntervalOld  = 40 * time.Minute

	// A pet sitting in its own mess for too long gets sick.
	sickAfterPoop = 30 * time.Minute

	// How long a call for attention lasts.
	callDuration = 15 * time.Minute

	// How long the device waits to see whether a second button is joining
	// the first one.
	comboWindow = 120 * time.Millisecond

	// The pet sleeps at night.
	sleepHour = 21
	wakeHour  = 7
)

// Event is something the pet wants the outside world to know about.
type Event struct {
	Freq uint32
	Dur  uint32
}

// Pet is the state of the creature.
type Pet struct {
	Stage      Stage
	Age        int
	Weight     int
	Hunger     int
	Happy      int
	Discipline int
	Sick       bool
	Poops      int
	Sound      bool
	Light      bool
	Asleep     bool
	Calling    bool
	CallReason Reason
	CallUntil  time.Time

	// What is on screen.
	Selected  int
	View      View
	Choice    int
	MeterPage int

	// The mini game.
	GameTarget  int
	GameRound   int
	GameCorrect int

	// Transient state, cleared when its time is up.
	Mood         Mood
	MoodUntil    time.Time
	FlashUntil   time.Time
	LastHunger   time.Time
	LastHappy    time.Time
	LastAge      time.Time
	LastPoop     time.Time
	LastPoopSeen time.Time
	AsleepSince  time.Time
	WokeAt       time.Time

	// A small generator so the pet's whole state can be saved.
	RNG rng

	// The time the pet last saw. Kept so that the pet can move its own
	// clock forward without reaching for the wall clock.
	now time.Time

	// The state of the three buttons, and the pending A press that waits
	// to see whether a second button is joining it.
	buttons      [3]bool
	pendingA     bool
	pendingUntil time.Time

	// The pet's own idea of the time, which it can move with the clock
	// screen.
	ClockOffset time.Duration
	SetMode     bool

	// Events waiting to be picked up by the engine.
	Outbox []Event
}

// Mood is the expression on the pet's face.
type Mood int

// The moods.
const (
	MoodIdle Mood = iota
	MoodHappy
	MoodSad
	MoodEating
	MoodPlaying
	MoodSick
	MoodDead
	MoodHatching
)

// New returns a pet that has just hatched.
func New(now time.Time) *Pet {
	p := &Pet{
		Stage:      StageBaby,
		Weight:     20,
		Hunger:     4,
		Happy:      4,
		Sound:      true,
		Light:      true,
		Selected:   -1,
		LastHunger: now,
		LastHappy:  now,
		LastAge:    now,
		LastPoop:   now,
		WokeAt:     now,
		now:        now,
	}
	p.RNG.state = 0x2545F4914F6CDD1D
	return p
}

// rng is a linear congruential generator.
//
// The pet needs randomness but its entire state has to survive a save and
// restore, so it brings its own rather than leaning on the global one.
type rng struct {
	state uint64
}

func (r *rng) next() uint64 {
	r.state = r.state*6364136223846793005 + 1442695040888963407
	return r.state >> 33
}

func (r *rng) intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.next() % uint64(n))
}

// Emit queues a sound for the engine to play.
//
// A muted pet keeps its mouth shut.
func (p *Pet) Emit(freq, dur uint32) {
	if !p.Sound {
		return
	}
	p.Outbox = append(p.Outbox, Event{Freq: freq, Dur: dur})
}

// Drain takes the queued sounds.
func (p *Pet) Drain() []Event {
	if len(p.Outbox) == 0 {
		return nil
	}
	out := p.Outbox
	p.Outbox = nil
	return out
}

// SetButtons records the state of the three buttons and acts on whatever
// changed.
//
// The original device acts on the press, not the release. It also has one
// combination, A with C, which mutes the sound and which is only meant to
// happen on the main screen. So a press of A there waits a moment to see
// whether C is joining it.
func (p *Pet) SetButtons(a, b, c bool, now time.Time) {
	state := [3]bool{a, b, c}

	var pressed [3]bool
	for i := range state {
		pressed[i] = state[i] && !p.buttons[i]
	}
	p.buttons = state

	if p.pendingA {
		switch {
		case pressed[2]:
			// C joined A, so this is the combination.
			p.pendingA = false
			p.toggleSound()
			return
		case !state[0]:
			// A came and went without C joining it, so it was an ordinary
			// press. Acting now rather than at the end of the window keeps
			// a quick tap feeling immediate.
			p.pendingA = false
			p.Press(ButtonA, now)
		case !now.Before(p.pendingUntil):
			p.pendingA = false
			p.Press(ButtonA, now)
		}
	}

	// Both buttons can arrive in the same frame, which is what happens when
	// the presses are closer together than the frame rate. On the main
	// screen that is the sound combination and nothing else.
	if pressed[0] && pressed[2] && p.View == ViewMain && p.Selected < 0 {
		p.toggleSound()
		return
	}

	if pressed[0] && p.View == ViewMain && p.Selected < 0 {
		p.pendingA = true
		p.pendingUntil = now.Add(comboWindow)
		return
	}

	for i := range pressed {
		if pressed[i] {
			p.Press(Button(i), now)
		}
	}
}

// toggleSound mutes or unmutes the pet.
func (p *Pet) toggleSound() {
	p.Sound = !p.Sound
	if p.Sound {
		p.Emit(5200, 60)
	}
}

// Press handles a button being pressed.
//
// The original device acts on the press, not the release, so this is called
// once per press.
func (p *Pet) Press(b Button, now time.Time) {
	p.now = now

	if p.Stage == StageEgg {
		// The egg does nothing but sit there until it hatches.
		return
	}
	if p.Dead() {
		return
	}

	switch p.View {
	case ViewMain:
		p.pressMain(b, now)
	case ViewFeed:
		p.pressFeed(b)
	case ViewLight:
		p.pressLight(b)
	case ViewPlay:
		p.pressPlay(b, now)
	case ViewMedicine:
		p.pressMedicine(b, now)
	case ViewDuck:
		p.pressDuck(b, now)
	case ViewMeter:
		p.pressMeter(b)
	case ViewDiscipline:
		p.pressDiscipline(b, now)
	case ViewClock:
		p.pressClock(b, now)
	}
}

func (p *Pet) pressMain(b Button, now time.Time) {
	switch b {
	case ButtonA:
		if p.Selected < 0 {
			p.Selected = 0
			p.Emit(4400, 40)
			return
		}
		p.Selected = (p.Selected + 1) % len(MenuIcons)
		p.Emit(4400, 40)
	case ButtonB:
		if p.Selected < 0 {
			// With nothing selected the middle button brings up the
			// clock, which is what the original does.
			p.View = ViewClock
			p.SetMode = false
			return
		}
		p.openSelected()
	case ButtonC:
		if p.Selected < 0 {
			return
		}
		p.Selected = -1
		p.Emit(3200, 40)
	}
}

// openSelected opens the view for the highlighted icon.
func (p *Pet) openSelected() {
	switch MenuIcons[p.Selected] {
	case lcd.IconFeed:
		p.View = ViewFeed
		p.Choice = 0
	case lcd.IconLight:
		p.View = ViewLight
		if p.Light {
			p.Choice = 1
		} else {
			p.Choice = 0
		}
	case lcd.IconPlay:
		p.View = ViewPlay
		p.GameRound = 0
		p.GameCorrect = 0
		p.GameTarget = p.RNG.intn(2)
		p.Mood = MoodPlaying
		p.MoodUntil = nowPlus(p, 2*time.Second)
	case lcd.IconMedicine:
		p.View = ViewMedicine
	case lcd.IconDuck:
		p.View = ViewDuck
	case lcd.IconMeter:
		p.View = ViewMeter
		p.MeterPage = 0
	case lcd.IconDiscipline:
		p.View = ViewDiscipline
	}
}

// nowPlus is a helper so the pet can move its own clock forward without
// reaching for the wall clock.
func nowPlus(p *Pet, d time.Duration) time.Time {
	return p.now.Add(d)
}

func (p *Pet) pressFeed(b Button) {
	switch b {
	case ButtonA:
		p.Choice = 1 - p.Choice
	case ButtonB:
		if p.Choice == 0 {
			// A meal fills one heart of hunger.
			if p.Hunger < 4 {
				p.Hunger++
				p.Weight++
				p.Mood = MoodEating
			} else {
				// A full pet refuses to eat.
				p.Mood = MoodSad
			}
		} else {
			// A snack fills one heart of happiness, and weighs more.
			if p.Happy < 4 {
				p.Happy++
				p.Weight += 2
				p.Mood = MoodHappy
			}
		}
		p.MoodUntil = p.now.Add(2 * time.Second)
		p.View = ViewMain
		p.Selected = -1
	case ButtonC:
		p.View = ViewMain
		p.Selected = -1
	}
}

func (p *Pet) pressLight(b Button) {
	switch b {
	case ButtonA:
		p.Choice = 1 - p.Choice
	case ButtonB:
		p.Light = p.Choice == 0
		p.View = ViewMain
		p.Selected = -1
	case ButtonC:
		p.View = ViewMain
		p.Selected = -1
	}
}

func (p *Pet) pressPlay(b Button, now time.Time) {
	switch b {
	case ButtonA, ButtonB:
		// The original game asks which way the pet will turn. The two
		// outer answers are left and right.
		guess := 0
		if b == ButtonB {
			guess = 1
		}
		if guess == p.GameTarget {
			p.GameCorrect++
			p.Mood = MoodHappy
		} else {
			p.Mood = MoodSad
		}
		p.GameRound++
		if p.GameRound >= 5 {
			p.finishGame()
			return
		}
		p.GameTarget = p.RNG.intn(2)
		p.Mood = MoodPlaying
		p.MoodUntil = nowPlus(p, 2*time.Second)
	case ButtonC:
		// The right button stops the game.
		p.View = ViewMain
		p.Selected = -1
		p.Mood = MoodIdle
	}
}

// finishGame settles the score of the mini game.
//
// Five rounds, and at least three right wins a heart.
func (p *Pet) finishGame() {
	if p.GameCorrect >= 3 {
		if p.Happy < 4 {
			p.Happy++
		}
		p.Mood = MoodHappy
	} else {
		if p.Happy > 0 {
			p.Happy--
		}
		p.Mood = MoodSad
	}
	p.MoodUntil = nowPlus(p, 2*time.Second)
	p.View = ViewMain
	p.Selected = -1
}

func (p *Pet) pressMedicine(b Button, now time.Time) {
	switch b {
	case ButtonB:
		if p.Sick {
			p.Sick = false
			p.Mood = MoodHappy
			p.MoodUntil = nowPlus(p, 2*time.Second)
		}
		p.View = ViewMain
		p.Selected = -1
	case ButtonC:
		p.View = ViewMain
		p.Selected = -1
	}
}

func (p *Pet) pressDuck(b Button, now time.Time) {
	switch b {
	case ButtonB:
		p.Poops = 0
		p.Mood = MoodHappy
		p.MoodUntil = nowPlus(p, 2*time.Second)
		p.View = ViewMain
		p.Selected = -1
	case ButtonC:
		p.View = ViewMain
		p.Selected = -1
	}
}

func (p *Pet) pressMeter(b Button) {
	switch b {
	case ButtonA, ButtonB:
		p.MeterPage = (p.MeterPage + 1) % 4
	case ButtonC:
		p.View = ViewMain
		p.Selected = -1
	}
}

func (p *Pet) pressDiscipline(b Button, now time.Time) {
	switch b {
	case ButtonB:
		// Scolding only works when the pet is calling for no reason. If it
		// genuinely needs something, being scolded does nothing.
		if p.Calling && p.CallReason == ReasonNaughty {
			p.Discipline++
			if p.Discipline > 4 {
				p.Discipline = 4
			}
			p.Mood = MoodSad
			p.MoodUntil = nowPlus(p, 2*time.Second)
			p.clearCall()
		}
		p.View = ViewMain
		p.Selected = -1
	case ButtonC:
		p.View = ViewMain
		p.Selected = -1
	}
}

// pressClock handles the clock screen.
//
// With A held down, C starts setting the time. A moves the hours, B moves
// the minutes, and C confirms.
func (p *Pet) pressClock(b Button, now time.Time) {
	switch b {
	case ButtonA:
		if p.SetMode {
			p.ClockOffset += time.Hour
		}
	case ButtonB:
		if p.SetMode {
			p.ClockOffset += time.Minute
		} else {
			p.View = ViewMain
		}
	case ButtonC:
		if p.buttons[0] && !p.SetMode {
			// A is being held down, so this is the start of setting
			// the time rather than a way out of the screen.
			p.SetMode = true
			return
		}
		p.SetMode = false
		p.View = ViewMain
	}
}

// clearCall stops the pet calling for attention.
func (p *Pet) clearCall() {
	p.Calling = false
	p.CallReason = ReasonNone
}

// Dead reports whether the pet has died of neglect.
func (p *Pet) Dead() bool {
	return p.Hunger == 0 && p.Happy == 0 && p.Age > 0
}

// Tick advances the simulation.
//
// The engine calls this often. The pet works out how much time has passed
// and applies the rules for it.
func (p *Pet) Tick(now time.Time) {
	p.now = now

	if p.Mood != MoodIdle && !p.MoodUntil.IsZero() && now.After(p.MoodUntil) {
		p.Mood = MoodIdle
	}

	if p.Stage == StageEgg {
		return
	}

	p.updateSleep(now)
	p.updateHunger(now)
	p.updateHappy(now)
	p.updateAge(now)
	p.updatePoop(now)
	p.updateCall(now)
}

// updateSleep puts the pet to bed at night and wakes it in the morning.
func (p *Pet) updateSleep(now time.Time) {
	hour := now.Hour()
	shouldSleep := hour >= sleepHour || hour < wakeHour

	switch {
	case shouldSleep && !p.Asleep:
		p.Asleep = true
		p.AsleepSince = now
		p.Mood = MoodIdle
	case !shouldSleep && p.Asleep:
		p.Asleep = false
		p.WokeAt = now
		// The original counts a year every time the pet wakes up.
		p.Age++
		p.LastAge = now
		p.updateStage()
	}
}

// updateHunger takes one heart away every hunger interval.
func (p *Pet) updateHunger(now time.Time) {
	if p.Asleep {
		p.LastHunger = now
		return
	}
	for now.Sub(p.LastHunger) >= hungerInterval {
		p.LastHunger = p.LastHunger.Add(hungerInterval)
		if p.Hunger > 0 {
			p.Hunger--
		}
	}
}

// updateHappy takes one heart away every happy interval.
func (p *Pet) updateHappy(now time.Time) {
	if p.Asleep {
		p.LastHappy = now
		return
	}
	for now.Sub(p.LastHappy) >= happyInterval {
		p.LastHappy = p.LastHappy.Add(happyInterval)
		if p.Happy > 0 {
			p.Happy--
		}
	}
}

// updateAge counts a year every age interval.
func (p *Pet) updateAge(now time.Time) {
	for now.Sub(p.LastAge) >= ageInterval {
		p.LastAge = p.LastAge.Add(ageInterval)
		p.Age++
		p.updateStage()
	}
}

// updateStage moves the pet on as it grows.
func (p *Pet) updateStage() {
	switch {
	case p.Age >= 4:
		p.Stage = StageAdult
	case p.Age >= 3:
		p.Stage = StageTeen
	case p.Age >= 2:
		p.Stage = StageChild
	case p.Age >= 1:
		p.Stage = StageBaby
	}
}

// updatePoop makes a mess now and then, and lets the pet get sick if the
// mess is left alone.
func (p *Pet) updatePoop(now time.Time) {
	if p.Asleep {
		return
	}

	interval := poopIntervalBaby
	if p.Stage >= StageTeen {
		interval = poopIntervalOld
	}

	for now.Sub(p.LastPoop) >= interval {
		p.LastPoop = p.LastPoop.Add(interval)
		if p.Poops < 4 {
			p.Poops++
		}
	}

	if p.Poops > 0 && now.Sub(p.LastPoopSeen) >= sickAfterPoop {
		if !p.Sick && p.RNG.intn(4) == 0 {
			p.Sick = true
		}
	}
}

// updateCall makes the pet call for attention when it needs something.
func (p *Pet) updateCall(now time.Time) {
	if p.Calling {
		if now.After(p.CallUntil) {
			p.clearCall()
		}
		return
	}

	reason := ReasonNone
	switch {
	case p.Hunger == 0:
		reason = ReasonHungry
	case p.Happy == 0:
		reason = ReasonUnhappy
	case p.Asleep && p.Light:
		reason = ReasonLight
	case p.Dead():
		reason = ReasonNone
	}

	if reason == ReasonNone {
		// A pet with nothing wrong sometimes calls anyway. Being disciplined
		// stops it.
		if p.Hunger > 0 && p.Happy > 0 && !p.Asleep && p.Discipline < 2 && p.RNG.intn(20) == 0 {
			reason = ReasonNaughty
		}
	}

	if reason != ReasonNone {
		p.Calling = true
		p.CallReason = reason
		p.CallUntil = now.Add(callDuration)
	}
}
