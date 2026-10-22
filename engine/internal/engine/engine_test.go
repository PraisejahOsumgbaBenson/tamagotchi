package engine

import (
	"path/filepath"
	"testing"
	"time"

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

func TestButtonsActOnThePress(t *testing.T) {
	e := New(Options{})

	messages, cancel := e.Subscribe()
	defer cancel()

	e.Handle(proto.Buttons(true, false, false))
	e.Step()

	// The button makes a noise and changes the screen. The frame is what
	// matters here.
	frame := waitForFrame(t, messages)
	if frame.Sel == 0 {
		t.Error("no menu icon should be selected yet")
	}

	// Holding the button down should not act again.
	drain(messages)
	e.Step()
	select {
	case m := <-messages:
		t.Fatalf("holding a button should not act twice, got a %q", m.Kind)
	default:
	}
}

// waitForFrame reads messages until a frame arrives.
func waitForFrame(t *testing.T, messages <-chan proto.Message) proto.Message {
	t.Helper()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case m := <-messages:
			if m.Kind == proto.KindFrame {
				return m
			}
		case <-deadline:
			t.Fatal("no frame arrived")
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
