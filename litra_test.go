package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

// unhex parses "11 ff 06 4c" and pads it to a full report.
func unhex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatal(err)
	}
	return append(b, make([]byte, reportLen-len(b))...)
}

func TestReport(t *testing.T) {
	tests := []struct {
		name string
		got  []byte
		want string
	}{
		{"front power on", report(featFront, fnFrontSetPower, 1), "11 ff 06 1c 01"},
		{"front brightness 100 lm", report(featFront, fnFrontSetBrightness, 0x00, 0x64), "11 ff 06 4c 00 64"},
		{"front temperature 4000 K", report(featFront, fnFrontSetTemp, 0x0f, 0xa0), "11 ff 06 9c 0f a0"},
		{"front power query", report(featFront, fnFrontGetPower), "11 ff 06 01"},
		{"back power on", report(featBack, fnBackSetPower, 1), "11 ff 0a 4b 01"},
		{"back brightness 50%", report(featBack, fnBackSetBrightness, 0x00, 50), "11 ff 0a 2b 00 32"},
		{"back commit", report(featBackColor, fnBackCommit, 0, 0, 1), "11 ff 0c 7b 00 00 01"},
		{"back zone 3 red", zoneReport(3, 0xff, 0, 0), "11 ff 0c 1b 03 ff 01 01 ff 00 00 00 ff 00 00 00 ff 00 00 00"},
	}
	for _, tt := range tests {
		if want := unhex(t, tt.want); !bytes.Equal(tt.got, want) {
			t.Errorf("%s: got % x, want % x", tt.name, tt.got, want)
		}
	}
}

// Replies captured from the light over Bluetooth.
func TestParseReplies(t *testing.T) {
	if got := u16(unhex(t, "11 ff 06 31 00 c8")); got != 200 {
		t.Errorf("brightness: got %d, want 200", got)
	}
	if got := u16(unhex(t, "11 ff 06 81 10 68")); got != 4200 {
		t.Errorf("temperature: got %d, want 4200", got)
	}
	if got := u16(unhex(t, "11 ff 0a 1b 00 32")); got != 50 {
		t.Errorf("back brightness: got %d, want 50", got)
	}
}

func TestHidppError(t *testing.T) {
	if err := hidppError(unhex(t, "11 ff 06 31 00 c8")); err != nil {
		t.Errorf("normal reply: unexpected error %v", err)
	}
	if err := hidppError(unhex(t, "11 ff ff 06 4c 02")); err == nil {
		t.Error("error reply: got nil, want error")
	}
}

func TestBrightnessMapping(t *testing.T) {
	for pct, lm := range map[int]int{0: 30, 46: 200, 100: 400, -5: 30, 150: 400} {
		if got := lumenFromPercent(pct); got != lm {
			t.Errorf("lumenFromPercent(%d) = %d, want %d", pct, got, lm)
		}
	}
	for lm, pct := range map[int]int{30: 0, 200: 46, 400: 100} {
		if got := percentFromLumen(lm); got != pct {
			t.Errorf("percentFromLumen(%d) = %d, want %d", lm, got, pct)
		}
	}
	// A relative step must not drift: every percent survives the round trip.
	for pct := 0; pct <= 100; pct++ {
		if got := percentFromLumen(lumenFromPercent(pct)); got != pct {
			t.Errorf("round trip of %d%% gave %d%%", pct, got)
		}
	}
}

func TestRoundTemp(t *testing.T) {
	for k, want := range map[int]int{4200: 4200, 4249: 4200, 4250: 4300, 1000: 2700, 9000: 6500} {
		if got := roundTemp(k); got != want {
			t.Errorf("roundTemp(%d) = %d, want %d", k, got, want)
		}
	}
}

func TestResolve(t *testing.T) {
	current := func() (int, error) { return 46, nil }
	for arg, want := range map[string]int{"80": 80, "+5": 51, "-5": 41} {
		got, err := resolve(arg, current)
		if err != nil || got != want {
			t.Errorf("resolve(%q) = %d, %v; want %d", arg, got, err, want)
		}
	}
	for _, arg := range []string{"", "abc", "5x"} {
		if _, err := resolve(arg, current); !errors.Is(err, errUsage) {
			t.Errorf("resolve(%q): got %v, want usage error", arg, err)
		}
	}
}

func TestParsePicked(t *testing.T) {
	r, g, b, err := parsePicked("65535, 27242, 0\n")
	if err != nil || r != 0xff || g != 0x6a || b != 0x00 {
		t.Errorf("parsePicked = %02x %02x %02x, %v", r, g, b, err)
	}
	if _, _, _, err := parsePicked(""); err == nil {
		t.Error("parsePicked of empty output: got nil error")
	}
}

func TestParseColor(t *testing.T) {
	for _, s := range []string{"ff6a00", "#FF6A00"} {
		r, g, b, err := parseColor(s)
		if err != nil || r != 0xff || g != 0x6a || b != 0x00 {
			t.Errorf("parseColor(%q) = %02x %02x %02x, %v", s, r, g, b, err)
		}
	}
	for _, s := range []string{"", "fff", "gggggg", "ff6a001"} {
		if _, _, _, err := parseColor(s); err == nil {
			t.Errorf("parseColor(%q): got nil error", s)
		}
	}
}
