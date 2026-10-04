package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestHidID(t *testing.T) {
	usb := "DRIVER=hid-generic\nHID_ID=0003:0000046D:0000C903\nHID_NAME=Logitech Litra Beam LX\n"
	if bus, vendor, product, ok := hidID(usb); !ok || bus != 3 || vendor != vendorID || product != productUSB {
		t.Errorf("USB: got %x %x %x %v", bus, vendor, product, ok)
	}
	ble := "HID_ID=0005:0000046D:0000B903\n"
	if bus, vendor, product, ok := hidID(ble); !ok || bus != 5 || vendor != vendorID || product != productBLE {
		t.Errorf("Bluetooth: got %x %x %x %v", bus, vendor, product, ok)
	}
	for _, uevent := range []string{"", "DRIVER=hid-generic\n", "HID_ID=nonsense\n"} {
		if _, _, _, ok := hidID(uevent); ok {
			t.Errorf("hidID(%q): got ok", uevent)
		}
	}
}

// fakeHidraw adds a hidraw device to a fake /sys/class/hidraw tree.
func fakeHidraw(t *testing.T, name, id string, vendorPage bool) {
	t.Helper()
	dir := filepath.Join(hidrawClass, name, "device")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	descriptor := []byte{0x05, 0x0c, 0x09, 0x01, 0xa1, 0x01, 0xc0}
	if vendorPage {
		descriptor = append(descriptor, 0x06, 0x43, 0xff, 0x0a, 0x02, 0x02, 0xa1, 0x01, 0xc0)
	}
	files := map[string][]byte{
		"uevent":            []byte("DRIVER=hid-generic\nHID_ID=" + id + "\nHID_NAME=test\n"),
		"report_descriptor": descriptor,
	}
	for file, content := range files {
		if err := os.WriteFile(filepath.Join(dir, file), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFindLight(t *testing.T) {
	realClass := hidrawClass
	hidrawClass = t.TempDir()
	t.Cleanup(func() { hidrawClass = realClass })

	expect := func(step, want string) {
		t.Helper()
		got, err := findLight()
		if want == "" && err == nil {
			t.Errorf("%s: found %s, want an error", step, got)
		}
		if want != "" && (err != nil || got != want) {
			t.Errorf("%s: got %q, %v; want %q", step, got, err, want)
		}
	}

	expect("nothing connected", "")

	// An MX mouse has the same vendor collection but is not the light.
	fakeHidraw(t, "hidraw0", "0005:0000046D:0000B042", true)
	fakeHidraw(t, "hidraw1", "0005:000004F2:0000B903", true) // another vendor
	expect("only other devices", "")

	fakeHidraw(t, "hidraw2", "0005:0000046D:0000B903", false)
	expect("an interface of the light without the vendor collection", "")

	fakeHidraw(t, "hidraw3", "0005:0000046D:0000B903", true)
	expect("Bluetooth", "/dev/hidraw3")

	fakeHidraw(t, "hidraw4", "0003:0000046D:0000C903", true)
	expect("USB and Bluetooth: USB wins", "/dev/hidraw4")

	fakeHidraw(t, "hidraw5", "0005:0000046D:0000B903", true)
	expect("USB still wins when a Bluetooth node sorts after it", "/dev/hidraw4")
}

// TestRequest talks to a stand-in for the light over a socket pair that, like
// hidraw, keeps one report per read.
func TestRequest(t *testing.T) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_SEQPACKET, 0)
	if err != nil {
		t.Fatal(err)
	}
	ours := os.NewFile(uintptr(fds[0]), "light")
	lamp := os.NewFile(uintptr(fds[1]), "lamp")
	dev = &light{file: ours, reports: make(chan []byte, 64)}
	go dev.readReports()
	t.Cleanup(func() {
		dev = nil
		ours.Close()
		lamp.Close()
	})

	// Answers every request except temperature queries, after two reports
	// that have nothing to do with it.
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := lamp.Read(buf)
			if err != nil {
				return
			}
			out := buf[:n]
			if out[2] == featFront && out[3] == fnFrontGetTemp {
				continue
			}
			lamp.Write([]byte{0x01, 0x00}) // consumer-control report
			lamp.Write(report(0x05, 0x00)) // HID++ report for someone else
			lamp.Write(report(out[2], out[3], 0x00, 0xc8))
		}
	}()

	for range 3 { // repeats also exercise dropping the leftover reports
		reply, err := request(report(featFront, fnFrontGetBrightness))
		if err != nil || u16(reply) != 200 {
			t.Fatalf("answered request: got % x, %v", reply, err)
		}
	}

	if _, err := request(report(featFront, fnFrontGetTemp)); err == nil || !strings.Contains(err.Error(), "no reply") {
		t.Errorf("unanswered request: got %v, want a no-reply error", err)
	}

	lamp.Close()
	if _, err := request(report(featFront, fnFrontGetBrightness)); err == nil {
		t.Error("light gone: got no error")
	}
}

func TestParsePicked(t *testing.T) {
	for _, s := range []string{"rgb(255,106,0)", "rgba(255,106,0,1)", "#ff6a00", "#FF6A00", "#ffff6a6a0000"} {
		r, g, b, err := parsePicked(s)
		if err != nil || r != 0xff || g != 0x6a || b != 0x00 {
			t.Errorf("parsePicked(%q) = %02x %02x %02x, %v", s, r, g, b, err)
		}
	}
	for _, s := range []string{"", "rgb", "rgb(1,2)", "rgb(300,0,0)", "#ff6a0", "#gggggg", "red"} {
		if _, _, _, err := parsePicked(s); err == nil {
			t.Errorf("parsePicked(%q): got nil error", s)
		}
	}
}
