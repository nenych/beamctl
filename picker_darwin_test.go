package main

import "testing"

func TestParsePicked(t *testing.T) {
	r, g, b, err := parsePicked("65535, 27242, 0\n")
	if err != nil || r != 0xff || g != 0x6a || b != 0x00 {
		t.Errorf("parsePicked = %02x %02x %02x, %v", r, g, b, err)
	}
	if _, _, _, err := parsePicked(""); err == nil {
		t.Error("parsePicked of empty output: got nil error")
	}
}
