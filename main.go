package main

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"runtime"
	"slices"
	"strconv"
)

const usage = `usage:
  beamctl status
  beamctl on | off | toggle
  beamctl brightness <0-100 | +N | -N>       percent of the 30-400 lm range
  beamctl temp <2700-6500 | +N | -N>         kelvin, rounded to 100
  beamctl back on | off | toggle
  beamctl back brightness <1-100 | +N | -N>  percent
  beamctl back color <RRGGBB>
  beamctl back pick                          choose in the system colour picker, turns the back light on
  beamctl preset [name]                      from ~/.config/beamctl/presets.json
`

var errUsage = errors.New("usage")

func main() {
	// The HID run loop belongs to a thread; keep every cgo call on this one.
	runtime.LockOSThread()

	err := run(os.Args[1:])
	if errors.Is(err, errUsage) {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "beamctl:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errUsage
	}
	cmd, rest := args[0], args[1:]
	switch {
	case cmd == "status" && len(rest) == 0:
		return status()
	case isPower(cmd) && len(rest) == 0:
		return power(featFront, fnFrontGetPower, fnFrontSetPower, cmd)
	case cmd == "brightness" && len(rest) == 1:
		pct, err := resolve(rest[0], func() (int, error) {
			lm, err := getU16(featFront, fnFrontGetBrightness)
			return percentFromLumen(lm), err
		})
		if err != nil {
			return err
		}
		return setFrontBrightness(pct)
	case cmd == "temp" && len(rest) == 1:
		k, err := resolve(rest[0], func() (int, error) {
			return getU16(featFront, fnFrontGetTemp)
		})
		if err != nil {
			return err
		}
		return setFrontTemp(k)
	case cmd == "back" && len(rest) == 1 && isPower(rest[0]):
		return power(featBack, fnBackGetPower, fnBackSetPower, rest[0])
	case cmd == "back" && len(rest) == 2 && rest[0] == "brightness":
		pct, err := resolve(rest[1], func() (int, error) {
			return getU16(featBack, fnBackGetBrightness)
		})
		if err != nil {
			return err
		}
		return setBackBrightness(pct)
	case cmd == "back" && len(rest) == 2 && rest[0] == "color":
		r, g, b, err := parseColor(rest[1])
		if err != nil {
			return err
		}
		return setBackColor(r, g, b)
	case cmd == "back" && len(rest) == 1 && rest[0] == "pick":
		r, g, b, ok, err := pickColor()
		if !ok {
			return err
		}
		if err := setBackColor(r, g, b); err != nil {
			return err
		}
		return setPower(featBack, fnBackSetPower, true)
	case cmd == "preset" && len(rest) <= 1:
		return runPreset(rest)
	}
	return errUsage
}

func isPower(arg string) bool { return arg == "on" || arg == "off" || arg == "toggle" }

func power(feature, get, set byte, arg string) error {
	on := arg == "on"
	if arg == "toggle" {
		cur, err := getBool(feature, get)
		if err != nil {
			return err
		}
		on = !cur
	}
	return setPower(feature, set, on)
}

// resolve turns "80", "+5" or "-5" into an absolute value, reading the
// current one only for a relative argument.
func resolve(arg string, current func() (int, error)) (int, error) {
	n, err := strconv.Atoi(arg)
	if err != nil {
		return 0, errUsage
	}
	if arg[0] != '+' && arg[0] != '-' {
		return n, nil
	}
	cur, err := current()
	return cur + n, err
}

func status() error {
	frontOn, err := getBool(featFront, fnFrontGetPower)
	if err != nil {
		return err
	}
	lm, err := getU16(featFront, fnFrontGetBrightness)
	if err != nil {
		return err
	}
	k, err := getU16(featFront, fnFrontGetTemp)
	if err != nil {
		return err
	}
	backOn, err := getBool(featBack, fnBackGetPower)
	if err != nil {
		return err
	}
	backPct, err := getU16(featBack, fnBackGetBrightness)
	if err != nil {
		return err
	}
	fmt.Printf("front: %-3s  %d%% (%d lm)  %d K\n", onOff(frontOn), percentFromLumen(lm), lm, k)
	fmt.Printf("back:  %-3s  %d%%\n", onOff(backOn), backPct)
	return nil
}

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

// runPreset lists preset names, one per line, or applies the named one.
func runPreset(args []string) error {
	presets, err := loadPresets()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		for _, name := range slices.Sorted(maps.Keys(presets)) {
			fmt.Println(name)
		}
		return nil
	}
	p, ok := presets[args[0]]
	if !ok {
		return fmt.Errorf("unknown preset %q", args[0])
	}
	return applyPreset(p)
}
