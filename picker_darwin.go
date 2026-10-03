package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// parsePicked decodes the result of osascript's "choose color": three 16-bit channels.
func parsePicked(out string) (r, g, b byte, err error) {
	var r16, g16, b16 uint16
	if _, err := fmt.Sscanf(out, "%d, %d, %d", &r16, &g16, &b16); err != nil {
		return 0, 0, 0, fmt.Errorf("unexpected colour picker output %q", out)
	}
	return byte(r16 >> 8), byte(g16 >> 8), byte(b16 >> 8), nil
}

// pickColor shows the macOS colour picker; ok is false when it was cancelled.
// It starts from white: the default is black, which puts the brightness
// slider at zero and renders the whole colour wheel black.
func pickColor() (r, g, b byte, ok bool, err error) {
	out, err := exec.Command("osascript", "-e", "choose color default color {65535, 65535, 65535}").Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && strings.Contains(string(exit.Stderr), "-128") {
		return 0, 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, 0, false, err
	}
	r, g, b, err = parsePicked(string(out))
	return r, g, b, err == nil, err
}
