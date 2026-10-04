package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	vendorID   = 0x046d
	productUSB = 0xc903
	productBLE = 0xb903
)

// vendorPageItem is the report descriptor item "Usage Page (0xFF43)", which
// opens the Logitech vendor collection that carries HID++.
var vendorPageItem = []byte{0x06, 0x43, 0xff}

// Where the kernel lists hidraw devices and where their nodes live; variables
// so that tests can point them at a fake tree.
var (
	hidrawClass = "/sys/class/hidraw"
	devDir      = "/dev"
)

// Set BEAMCTL_DEBUG=1 to see which HID devices were found and why one was chosen.
var debug = os.Getenv("BEAMCTL_DEBUG") != ""

func debugf(format string, args ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "debug: "+format+"\n", args...)
	}
}

// hidID reads bus, vendor and product from the HID_ID line of a uevent file,
// for example "HID_ID=0003:0000046D:0000C903".
func hidID(uevent string) (bus, vendor, product uint32, ok bool) {
	for line := range strings.SplitSeq(uevent, "\n") {
		if id, found := strings.CutPrefix(line, "HID_ID="); found {
			_, err := fmt.Sscanf(id, "%x:%x:%x", &bus, &vendor, &product)
			return bus, vendor, product, err == nil
		}
	}
	return 0, 0, 0, false
}

// findLight returns the hidraw node of a Litra Beam LX connected over USB or
// Bluetooth. USB wins when the light is reachable both ways.
func findLight() (string, error) {
	nodes, err := filepath.Glob(filepath.Join(hidrawClass, "hidraw*"))
	if err != nil {
		return "", err
	}
	var found string
	for _, node := range nodes {
		uevent, err := os.ReadFile(filepath.Join(node, "device", "uevent"))
		if err != nil {
			continue
		}
		bus, vendor, product, ok := hidID(string(uevent))
		if !ok || vendor != vendorID {
			continue
		}
		// A device has one hidraw node per interface; take the one whose
		// report descriptor contains the vendor collection.
		descriptor, _ := os.ReadFile(filepath.Join(node, "device", "report_descriptor"))
		vendorPage := bytes.Contains(descriptor, vendorPageItem)
		path := filepath.Join(devDir, filepath.Base(node))
		debugf("Logitech HID: bus %04x, product %04x, vendor collection %v, %s", bus, product, vendorPage, path)
		if !vendorPage || (product != productUSB && product != productBLE) {
			continue
		}
		if found == "" || product == productUSB {
			found = path
		}
	}
	if found == "" {
		return "", errors.New("Litra Beam LX not found (is it connected via USB or Bluetooth?)")
	}
	debugf("using %s", found)
	return found, nil
}

type light struct {
	file    *os.File
	reports chan []byte // closed when reading fails; readErr then says why
	readErr error
}

var dev *light

func openLight() (*light, error) {
	path, err := findLight()
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if errors.Is(err, os.ErrPermission) {
		return nil, fmt.Errorf("cannot open the light: no permission for %s; see \"Linux: access to the light\" in the README", path)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot open the light: %w", err)
	}
	l := &light{file: file, reports: make(chan []byte, 64)}
	go l.readReports()
	return l, nil
}

func (l *light) readReports() {
	for {
		buf := make([]byte, 64)
		n, err := l.file.Read(buf)
		if err != nil {
			l.readErr = err
			close(l.reports)
			return
		}
		l.reports <- buf[:n]
	}
}

// readFailure is the error to return once reports has been closed.
func (l *light) readFailure() error {
	return fmt.Errorf("cannot read from the light: %w", l.readErr)
}

// request sends one 20-byte report and returns the light's reply to it,
// opening the device on first use.
func request(out []byte) ([]byte, error) {
	if dev == nil {
		l, err := openLight()
		if err != nil {
			return nil, err
		}
		dev = l
	}

	// Drop reports left over from earlier requests.
	for pending := true; pending; {
		select {
		case _, ok := <-dev.reports:
			if !ok {
				return nil, dev.readFailure()
			}
		default:
			pending = false
		}
	}

	if _, err := dev.file.Write(out); err != nil {
		return nil, fmt.Errorf("cannot send to the light: %w", err)
	}

	timeout := time.After(time.Second)
	for {
		select {
		case in, ok := <-dev.reports:
			if !ok {
				return nil, dev.readFailure()
			}
			if isReplyTo(out, in) {
				return in[:reportLen], nil
			}
		case <-timeout:
			return nil, errors.New("no reply from the light")
		}
	}
}
