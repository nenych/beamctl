package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const noPickerHint = "use `beamctl back color RRGGBB` instead"

// pickColor shows a colour dialog, starting from white, through zenity or
// kdialog, whichever is installed; ok is false when it was cancelled.
func pickColor() (r, g, b byte, ok bool, err error) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return 0, 0, 0, false, errors.New("no graphical session for a colour dialog; " + noPickerHint)
	}
	var cmd *exec.Cmd
	if path, err := exec.LookPath("zenity"); err == nil {
		cmd = exec.Command(path, "--color-selection", "--color=#ffffff")
	} else if path, err := exec.LookPath("kdialog"); err == nil {
		cmd = exec.Command(path, "--getcolor", "--default", "#ffffff")
	} else {
		return 0, 0, 0, false, errors.New("no colour dialog found: install zenity or kdialog, or " + noPickerHint)
	}
	out, err := cmd.Output()
	// Both dialogs exit with status 1 when cancelled.
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return 0, 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, 0, false, err
	}
	r, g, b, err = parsePicked(strings.TrimSpace(string(out)))
	return r, g, b, err == nil, err
}

// parsePicked decodes what the dialogs print: "rgb(255,106,0)" from zenity,
// "#ff6a00" from kdialog, or "#ffff6a6a0000" from older versions of zenity.
func parsePicked(s string) (r, g, b byte, err error) {
	switch {
	case strings.HasPrefix(s, "rgb"):
		var red, green, blue uint8
		if _, err := fmt.Sscanf(s[strings.Index(s, "(")+1:], "%d,%d,%d", &red, &green, &blue); err == nil {
			return red, green, blue, nil
		}
	case len(s) == 7 && s[0] == '#':
		return parseColor(s)
	case len(s) == 13 && s[0] == '#':
		return parseColor(s[1:3] + s[5:7] + s[9:11])
	}
	return 0, 0, 0, fmt.Errorf("unexpected colour dialog output %q", s)
}
