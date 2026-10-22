// Package engine drives the pet and produces the frames that go to the
// shell.
//
// The engine knows nothing about windows, pixels or sound. It runs happily
// with nobody looking at it, which is what makes it testable and what makes
// it survive the shell being restarted.
package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PraisejahOsumgbaBenson/tamagotchi/engine/internal/lcd"
	"github.com/PraisejahOsumgbaBenson/tamagotchi/engine/internal/pet"
	"github.com/PraisejahOsumgbaBenson/tamagotchi/engine/internal/proto"
)

// Options configure an Engine.
type Options struct {
	// Clock returns the current time. It defaults to time.Now.
	Clock func() time.Time

	// StatePath is where the save state is kept. An empty path turns
	// persistence off.
	StatePath string

	// FrameRate is how many frames per second the engine produces. It
	// defaults to the protocol's frame rate.
	FrameRate int
}

// Engine owns the pet.
//
// Everything the engine does happens under one lock. The work is tiny, the
// rate is low, and a single lock is easier to reason about than the
// alternative.
type Engine struct {
	mu     sync.Mutex
	pet    *pet.Pet
	screen lcd.Screen
	icons  uint32
	sel    uint32

	clock  func() time.Time
	offset time.Duration

	// buttons is the state the shell last reported, and pressed remembers
	// the buttons that went down since the last frame. Without the latch a
	// tap that starts and finishes between two frames is never seen at all.
	buttons [3]bool
	pressed [3]bool

	statePath string
	frameRate int

	subs map[chan proto.Message]struct{}
}

// New builds an engine.
func New(opts Options) *Engine {
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}

	e := &Engine{
		pet:       pet.New(clock()),
		clock:     clock,
		statePath: opts.StatePath,
		frameRate: opts.FrameRate,
		subs:      make(map[chan proto.Message]struct{}),
	}

	if opts.StatePath != "" {
		if err := e.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
			// A missing save file is normal. A corrupt one is not, but
			// refusing to start over it would be worse than starting from
			// a fresh pet, so it is reported and the pet begins anew.
			fmt.Fprintf(os.Stderr, "engine: could not read the save state: %v\n", err)
		}
	}

	return e
}

// Now returns the engine's idea of the time.
//
// The pet keeps its own offset on top of this, because the original lets
// you set the clock with the buttons.
func (e *Engine) Now() time.Time {
	e.mu.Lock()
	offset := e.pet.ClockOffset
	e.mu.Unlock()

	return e.clock().Add(e.offset).Add(offset)
}

// SetClock corrects the engine's clock. The original device keeps its own
// real time clock and the shell tells it what time it is.
func (e *Engine) SetClock(epoch int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.offset = time.Duration(epoch-e.clock().Unix()) * time.Second
}

// Subscribe returns a channel that receives every message the engine
// produces, and a function to stop receiving.
//
// The channel is closed when the subscription is cancelled.
func (e *Engine) Subscribe() (<-chan proto.Message, func()) {
	ch := make(chan proto.Message, 64)

	e.mu.Lock()
	e.subs[ch] = struct{}{}
	e.mu.Unlock()

	return ch, func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if _, ok := e.subs[ch]; ok {
			delete(e.subs, ch)
			close(ch)
		}
	}
}

// publish sends a message to every subscriber. A subscriber that cannot keep
// up misses the message rather than stalling the engine, because the next
// frame carries the same information.
func (e *Engine) publish(m proto.Message) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for ch := range e.subs {
		select {
		case ch <- m:
		default:
		}
	}
}

// Handle acts on a message from the shell.
func (e *Engine) Handle(m proto.Message) {
	switch m.Kind {
	case proto.KindButton:
		e.mu.Lock()
		next := [3]bool{m.A, m.B, m.C}
		for i := range next {
			if next[i] && !e.buttons[i] {
				e.pressed[i] = true
			}
		}
		e.buttons = next
		e.mu.Unlock()
	case proto.KindClock:
		e.SetClock(m.Epoch)
	case proto.KindSave:
		if err := e.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "engine: could not save the state: %v\n", err)
		}
	case proto.KindLoad:
		if err := e.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "engine: could not load the state: %v\n", err)
		}
	}
}

// Step advances the pet by one frame and publishes anything that changed.
//
// It is exported so that a test, or the headless mode, can drive the engine
// a frame at a time.
func (e *Engine) Step() {
	now := e.Now()

	e.mu.Lock()
	p := e.pet
	screen := e.screen

	// The pet works out what the buttons mean, including the
	// combinations, so it is told the whole state of all three. A button
	// that went down and came back up between frames is presented as still
	// down for this frame, and the next frame shows it released.
	a := e.buttons[0] || e.pressed[0]
	b := e.buttons[1] || e.pressed[1]
	c := e.buttons[2] || e.pressed[2]
	e.pressed = [3]bool{}

	p.SetButtons(a, b, c, now)

	p.Tick(now)

	icons, sel := p.Render(&screen, now)

	changed := !e.screen.Equal(screen) || e.icons != icons || e.sel != sel
	e.screen = screen
	e.icons = icons
	e.sel = sel

	var rows []uint32
	if changed {
		rows = screen.Rows()
	}
	events := p.Drain()
	e.mu.Unlock()

	if changed {
		e.publish(proto.Frame(rows, icons, sel))
	}
	for _, ev := range events {
		e.publish(proto.Audio(ev.Freq, ev.Dur))
	}
}

// Run drives the engine until the stop channel is closed.
func (e *Engine) Run(stop <-chan struct{}) {
	rate := proto.FrameRate
	if e.frameRate > 0 {
		rate = e.frameRate
	}
	interval := time.Second / time.Duration(rate)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			e.Step()
		}
	}
}

// Save writes the state to disk.
func (e *Engine) Save() error {
	path := e.statePath
	if path == "" {
		return errors.New("engine: no state path configured")
	}

	e.mu.Lock()
	data, err := json.MarshalIndent(e.pet, "", "  ")
	e.mu.Unlock()
	if err != nil {
		return fmt.Errorf("engine: encoding the state: %w", err)
	}

	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("engine: creating %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("engine: writing %s: %w", path, err)
	}
	return nil
}

// Load reads the state from disk.
func (e *Engine) Load() error {
	path := e.statePath
	if path == "" {
		return errors.New("engine: no state path configured")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	p := &pet.Pet{}
	if err := json.Unmarshal(data, p); err != nil {
		return fmt.Errorf("engine: parsing %s: %w", path, err)
	}

	e.mu.Lock()
	e.pet = p
	e.mu.Unlock()
	return nil
}

// Screen returns a copy of the current screen.
func (e *Engine) Screen() lcd.Screen {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.screen
}

// Snapshot returns the current screen as a frame.
//
// Frames are only published when something changes, so a shell that has just
// connected needs to be handed the current one rather than waiting for the
// pet to move.
func (e *Engine) Snapshot() proto.Message {
	e.mu.Lock()
	defer e.mu.Unlock()
	return proto.Frame(e.screen.Rows(), e.icons, e.sel)
}
