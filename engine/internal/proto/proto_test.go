package proto

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFrameRoundTrip(t *testing.T) {
	rows := make([]uint32, 16)
	for i := range rows {
		rows[i] = uint32(i * 7)
	}

	m := Frame(rows, 0b1011, 1<<2)
	data, err := Encode(m)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if got.Kind != KindFrame {
		t.Errorf("kind = %q, want %q", got.Kind, KindFrame)
	}
	if got.Icons != 0b1011 {
		t.Errorf("icons = %b, want %b", got.Icons, 0b1011)
	}
	if got.Sel != 1<<2 {
		t.Errorf("sel = %b, want %b", got.Sel, 1<<2)
	}
	if len(got.Rows) != 16 {
		t.Fatalf("got %d rows, want 16", len(got.Rows))
	}
	for i := range rows {
		if got.Rows[i] != rows[i] {
			t.Errorf("row %d = %d, want %d", i, got.Rows[i], rows[i])
		}
	}
}

func TestButtonCarriesTheWholeState(t *testing.T) {
	// A missing boolean means released, so a button message with only A set
	// means A is held and the others are not.
	data, err := json.Marshal(Buttons(true, false, false))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	m, err := Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !m.A || m.B || m.C {
		t.Errorf("got a=%v b=%v c=%v, want a=true b=false c=false", m.A, m.B, m.C)
	}
}

func TestClockRoundTrip(t *testing.T) {
	when := time.Unix(1790000000, 0)
	m := Clock(when)

	data, err := Encode(m)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Epoch != when.Unix() {
		t.Errorf("epoch = %d, want %d", got.Epoch, when.Unix())
	}
}

func TestDecodeRejectsNonsense(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"no kind", `{}`},
		{"unknown kind", `{"t":"nope"}`},
		{"short frame", `{"t":"frame","rows":[1,2,3]}`},
		{"silent audio", `{"t":"audio"}`},
		{"wrong version", `{"t":"hello","version":99}`},
		{"broken json", `{`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Decode([]byte(tc.in)); err == nil {
				t.Errorf("decoding %s should have failed", tc.in)
			}
		})
	}
}
