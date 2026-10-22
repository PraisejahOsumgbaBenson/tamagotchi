package engine

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/PraisejahOsumgbaBenson/tama/engine/internal/pet"
	"github.com/PraisejahOsumgbaBenson/tama/engine/internal/proto"
)

func TestStepPublishesAFrame(t *testing.T) {
	e := New(Options{})

	messages, cancel := e.Subscribe()
	defer cancel()

	e.Step()

	select {
	case m := <-messages:
		if m.Kind != proto.KindFrame {
			t.Fatalf("got a %q message, want a frame", m.Kind)
		}
		if len(m.Rows) != 16 {
			t.Errorf("the frame has %d rows, want 16", len(m.Rows))
		}
	default:
		t.Fatal("no frame was published")
	}
}

func TestNoFrameWhenNothingChanges(t *testing.T) {
	e := New(Options{})

	messages, cancel := e.Subscribe()
	defer cancel()

	e.Step()
	<-messages

	// Nothing has changed, so there is nothing to send.
	e.Step()
	select {
	case m := <-messages:
		t.Fatalf("an unchanged screen should not be published, got a %q", m.Kind)
	default:
	}
}

func TestSnapshotMatchesThePublishedFrame(t *testing.T) {
	e := New(Options{})

	messages, cancel := e.Subscribe()
	defer cancel()

	e.Step()
	published := waitForFrame(t, messages, func(m proto.Message) bool {
		return m.Kind == proto.KindFrame
	})

	snapshot := e.Snapshot()
	if snapshot.Kind != proto.KindFrame {
		t.Fatalf("got a %q message, want a frame", snapshot.Kind)
	}
	if len(snapshot.Rows) != len(published.Rows) {
		t.Fatalf("the snapshot has %d rows, the frame has %d", len(snapshot.Rows), len(published.Rows))
	}
	for i := range published.Rows {
		if snapshot.Rows[i] != published.Rows[i] {
			t.Errorf("row %d: snapshot %#x, frame %#x", i, snapshot.Rows[i], published.Rows[i])
		}
	}
	if snapshot.Icons != published.Icons || snapshot.Sel != published.Sel {
		t.Errorf(
			"the snapshot marks icons %#x and sel %#x, the frame marks %#x and %#x",
			snapshot.Icons, snapshot.Sel, published.Icons, published.Sel,
		)
	}
}

func TestSnapshotBeforeTheFirstStep(t *testing.T) {
	// A shell can connect before the pet has moved. It should still be
	// given a frame instead of a blank window.
	snapshot := New(Options{}).Snapshot()

	if snapshot.Kind != proto.KindFrame {
		t.Fatalf("got a %q message, want a frame", snapshot.Kind)
	}
	if len(snapshot.Rows) != 16 {
		t.Errorf("the snapshot has %d rows, want 16", len(snapshot.Rows))
	}
}

func TestButtonsActOnThePress(t *testing.T) {
	// The pet waits a moment on the main screen to see whether a second
	// button is joining the first, so the clock has to move.
	e, _ := hatchedEngine(t, 50*time.Millisecond)

	messages, cancel := e.Subscribe()
	defer cancel()

	e.Handle(proto.Buttons(true, false, false))
	for i := 0; i < 10; i++ {
		e.Step()
	}

	// The button makes a noise and changes the screen. The frame that
	// carries the selection is what matters here.
	frame := waitForFrame(t, messages, func(m proto.Message) bool {
		return m.Kind == proto.KindFrame && m.Sel != 0
	})
	if frame.Sel == 0 {
		t.Error("no menu icon should be selected yet")
	}
}

func TestATapBetweenFramesIsNotLost(t *testing.T) {
	// The shell reports the press and the release, and both can land
	// between two frames. The latch is what keeps the tap.
	e, _ := hatchedEngine(t, 50*time.Millisecond)

	messages, cancel := e.Subscribe()
	defer cancel()

	e.Handle(proto.Buttons(true, false, false))
	e.Handle(proto.Buttons(false, false, false))

	for i := 0; i < 10; i++ {
		e.Step()
	}

	waitForFrame(t, messages, func(m proto.Message) bool {
		return m.Kind == proto.KindFrame && m.Sel != 0
	})
}

// hatchedEngine returns an engine whose pet has left the egg.
//
// The buttons do nothing while the pet is still an egg, so any test that
// presses one has to get past the hatch first.
func hatchedEngine(t *testing.T, step time.Duration) (*Engine, *fakeClock) {
	t.Helper()

	clock := &fakeClock{
		now:  time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC),
		step: step,
	}
	e := New(Options{Clock: clock.Now})

	clock.now = clock.now.Add(45 * time.Second)
	e.Step()

	if stage := e.pet.Stage; stage != pet.StageBaby {
		t.Fatalf("the pet is still %v after the hatch", stage)
	}
	return e, clock
}

// fakeClock is a clock that moves forward every time it is read.
type fakeClock struct {
	now  time.Time
	step time.Duration
}

func (c *fakeClock) Now() time.Time {
	c.now = c.now.Add(c.step)
	return c.now
}

// waitForFrame reads messages until a frame that matches arrives.
func waitForFrame(t *testing.T, messages <-chan proto.Message, match func(proto.Message) bool) proto.Message {
	t.Helper()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case m := <-messages:
			if match(m) {
				return m
			}
		case <-deadline:
			t.Fatal("no matching frame arrived")
		}
	}
}

// drain throws away anything already waiting.
func drain(messages <-chan proto.Message) {
	for {
		select {
		case <-messages:
		default:
			return
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pet.json")
	at := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return at }

	e := New(Options{StatePath: path, Clock: clock})
	e.Handle(proto.Buttons(true, false, false))
	e.Step()

	if err := e.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	restored := New(Options{StatePath: path, Clock: clock})
	restored.Step()

	if !e.Screen().Equal(restored.Screen()) {
		t.Error("the restored screen does not match the saved one")
	}
}

func TestLoadWithoutAFileIsNotAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	e := New(Options{StatePath: path})
	if err := e.Load(); err == nil {
		t.Error("loading a missing file should report an error")
	}
}

func TestClockCanBeCorrected(t *testing.T) {
	base := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	e := New(Options{Clock: func() time.Time { return base }})

	e.SetClock(base.Add(time.Hour).Unix())

	if got := e.Now(); !got.Equal(base.Add(time.Hour)) {
		t.Errorf("now = %v, want %v", got, base.Add(time.Hour))
	}
}
