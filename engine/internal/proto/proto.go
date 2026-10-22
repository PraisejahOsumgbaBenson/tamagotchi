// Package proto defines the wire format spoken between the engine and the
// shell.
//
// The traffic is small and rare, so it is JSON text rather than binary. A
// person should be able to watch it go past in a terminal and understand it.
//
// Messages are a tagged union flattened onto a single object. The tag is the
// "t" field. Fields that do not belong to a message kind are omitted, so a
// frame carries rows and icons and nothing else.
package proto

import (
	"encoding/json"
	"fmt"
	"time"
)

// Kind tags a message.
type Kind string

// The message kinds.
const (
	KindHello  Kind = "hello"
	KindFrame  Kind = "frame"
	KindAudio  Kind = "audio"
	KindButton Kind = "btn"
	KindClock  Kind = "clock"
	KindSave   Kind = "save"
	KindLoad   Kind = "load"
)

// ProtocolVersion is bumped when the wire format changes in a way the shell
// has to know about. The shell refuses to talk to an engine it does not
// understand rather than showing a blank screen.
const ProtocolVersion = 1

// FrameRate is how many screen frames the engine produces per second. The
// original hardware refreshes its display at roughly this rate.
const FrameRate = 30

// Message is one item on the wire.
//
// Buttons are special: a button message always carries the full state of all
// three buttons rather than a delta. A missing boolean means released, so
// {"t":"btn","a":true} means A is held and B and C are not.
type Message struct {
	Kind Kind `json:"t"`

	// Frame.
	Rows  []uint32 `json:"rows,omitempty"`
	Icons uint32   `json:"icons,omitempty"`

	// Sel marks the menu icon that is currently highlighted. It is a
	// bitmask like Icons, and at most one bit is ever set.
	Sel uint32 `json:"sel,omitempty"`

	// Audio. Freq is in hertz and Dur is in milliseconds.
	Freq uint32 `json:"freq,omitempty"`
	Dur  uint32 `json:"dur,omitempty"`

	// Buttons.
	A bool `json:"a,omitempty"`
	B bool `json:"b,omitempty"`
	C bool `json:"c,omitempty"`

	// Clock. Epoch is seconds since the Unix epoch.
	Epoch int64 `json:"epoch,omitempty"`

	// Hello.
	Version int `json:"version,omitempty"`
}

// Hello greets a shell that has just connected.
func Hello() Message {
	return Message{Kind: KindHello, Version: ProtocolVersion}
}

// Frame carries one screen. Rows holds one bitmask per row of the 32 by 16
// display, Icons is a bitmask over the eight status icons, and Sel marks the
// menu icon that is highlighted.
func Frame(rows []uint32, icons, sel uint32) Message {
	return Message{Kind: KindFrame, Rows: rows, Icons: icons, Sel: sel}
}

// Audio asks the shell to sound the piezo.
func Audio(freqHz, durMS uint32) Message {
	return Message{Kind: KindAudio, Freq: freqHz, Dur: durMS}
}

// Buttons reports the state of all three buttons.
func Buttons(a, b, c bool) Message {
	return Message{Kind: KindButton, A: a, B: b, C: c}
}

// Clock tells the engine what time it is. The engine keeps its own idea of
// the time and uses this to correct it.
func Clock(t time.Time) Message {
	return Message{Kind: KindClock, Epoch: t.Unix()}
}

// Save asks the engine to write its state to disk.
func Save() Message {
	return Message{Kind: KindSave}
}

// Load asks the engine to restore its state from disk.
func Load() Message {
	return Message{Kind: KindLoad}
}

// Encode writes a message as JSON.
func Encode(m Message) ([]byte, error) {
	return json.Marshal(m)
}

// Decode reads a message from JSON.
func Decode(b []byte) (Message, error) {
	var m Message
	if err := json.Unmarshal(b, &m); err != nil {
		return Message{}, fmt.Errorf("proto: decode: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Message{}, err
	}
	return m, nil
}

// Validate checks that a message makes sense.
func (m Message) Validate() error {
	switch m.Kind {
	case KindHello:
		if m.Version != ProtocolVersion {
			return fmt.Errorf("proto: engine speaks version %d, this shell speaks %d", m.Version, ProtocolVersion)
		}
	case KindFrame:
		if len(m.Rows) != 16 {
			return fmt.Errorf("proto: frame has %d rows, want 16", len(m.Rows))
		}
	case KindAudio:
		if m.Freq == 0 {
			return fmt.Errorf("proto: audio event with no frequency")
		}
	case KindButton, KindClock, KindSave, KindLoad:
	case "":
		return fmt.Errorf("proto: message with no kind")
	default:
		return fmt.Errorf("proto: unknown message kind %q", m.Kind)
	}
	return nil
}
