package main

import (
	"fmt"
	"strconv"
	"strings"
)

// HID++ 2.0 feature indexes, as read from the Beam LX feature table.
const (
	featFront     = 0x06 // 0x1990 illumination
	featBack      = 0x0a // 0x8040 brightness control
	featBackColor = 0x0c // 0x8081 per-zone colour
)

// Function bytes (function nibble + software id) per feature.
const (
	fnFrontGetPower      = 0x01
	fnFrontSetPower      = 0x1c
	fnFrontGetBrightness = 0x31
	fnFrontSetBrightness = 0x4c
	fnFrontGetTemp       = 0x81
	fnFrontSetTemp       = 0x9c

	fnBackGetBrightness = 0x1b
	fnBackSetBrightness = 0x2b
	fnBackGetPower      = 0x3b
	fnBackSetPower      = 0x4b

	fnBackSetZone = 0x1b
	fnBackCommit  = 0x7b
)

const (
	reportLen = 20
	backZones = 7

	lumenMin, lumenMax = 30, 400
	tempMin, tempMax   = 2700, 6500
)

func report(feature, function byte, params ...byte) []byte {
	r := make([]byte, reportLen)
	r[0], r[1], r[2], r[3] = 0x11, 0xff, feature, function
	copy(r[4:], params)
	return r
}

// isReplyTo reports whether the input report answers the request out: either
// an echo of its feature and function bytes or a HID++ error naming them.
func isReplyTo(out, in []byte) bool {
	if len(in) < reportLen || in[0] != 0x11 {
		return false
	}
	echo := in[2] == out[2] && in[3] == out[3]
	failed := in[2] == 0xff && in[3] == out[2] && in[4] == out[3]
	return echo || failed
}

// hidppError decodes an error reply: 11 ff ff <feature> <function> <code>.
func hidppError(reply []byte) error {
	if reply[2] != 0xff {
		return nil
	}
	return fmt.Errorf("the light rejected the command (feature 0x%02x, function 0x%02x, error %d)",
		reply[3], reply[4], reply[5])
}

func call(feature, function byte, params ...byte) ([]byte, error) {
	reply, err := request(report(feature, function, params...))
	if err != nil {
		return nil, err
	}
	return reply, hidppError(reply)
}

func u16(reply []byte) int { return int(reply[4])<<8 | int(reply[5]) }

func getU16(feature, function byte) (int, error) {
	reply, err := call(feature, function)
	if err != nil {
		return 0, err
	}
	return u16(reply), nil
}

func getBool(feature, function byte) (bool, error) {
	reply, err := call(feature, function)
	if err != nil {
		return false, err
	}
	return reply[4] == 1, nil
}

func setPower(feature, function byte, on bool) error {
	var b byte
	if on {
		b = 1
	}
	_, err := call(feature, function, b)
	return err
}

func clamp(v, lo, hi int) int { return min(max(v, lo), hi) }

// Front brightness is exposed as 0-100% of the 30-400 lm range.
func lumenFromPercent(pct int) int {
	return lumenMin + (clamp(pct, 0, 100)*(lumenMax-lumenMin)+50)/100
}

func percentFromLumen(lm int) int {
	return ((clamp(lm, lumenMin, lumenMax)-lumenMin)*100 + (lumenMax-lumenMin)/2) / (lumenMax - lumenMin)
}

// roundTemp clamps to the supported range and rounds to the 100 K step the light accepts.
func roundTemp(k int) int {
	return (clamp(k, tempMin, tempMax) + 50) / 100 * 100
}

func setFrontBrightness(pct int) error {
	lm := lumenFromPercent(pct)
	_, err := call(featFront, fnFrontSetBrightness, byte(lm>>8), byte(lm))
	return err
}

func setFrontTemp(k int) error {
	k = roundTemp(k)
	_, err := call(featFront, fnFrontSetTemp, byte(k>>8), byte(k))
	return err
}

func setBackBrightness(pct int) error {
	_, err := call(featBack, fnBackSetBrightness, 0x00, byte(clamp(pct, 1, 100)))
	return err
}

func parseColor(s string) (r, g, b byte, err error) {
	s = strings.TrimPrefix(s, "#")
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil || len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid colour %q, want RRGGBB", s)
	}
	return byte(n >> 16), byte(n >> 8), byte(n), nil
}

// zoneReport sets one back zone. A zero channel can hang the light, so each is at least 1.
func zoneReport(zone, r, g, b byte) []byte {
	return report(featBackColor, fnBackSetZone, zone, max(r, 1), max(g, 1), max(b, 1),
		0xff, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0x00)
}

// setBackColor paints all zones, then commits once so they change together.
func setBackColor(r, g, b byte) error {
	for zone := byte(1); zone <= backZones; zone++ {
		reply, err := request(zoneReport(zone, r, g, b))
		if err != nil {
			return err
		}
		if err := hidppError(reply); err != nil {
			return err
		}
	}
	_, err := call(featBackColor, fnBackCommit, 0x00, 0x00, 0x01)
	return err
}
