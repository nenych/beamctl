package main

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <IOKit/hid/IOHIDLib.h>
#include <pthread.h>
#include <signal.h>
#include <stdlib.h>
#include <string.h>

#define REPORT_LEN 20
#define PID_USB 0xc903
#define PID_BLE 0xb903
#define NOT_FOUND 1
#define TIMEOUT 1

static IOHIDManagerRef mgr;
static IOHIDDeviceRef dev;
static uint8_t inbuf[64];
static uint8_t reply[REPORT_LEN];
static uint8_t want[2];
static int have;

// Keeps only the reply to the pending request: either an echo of its
// feature/function bytes or a HID++ error (ff <feature> <function> <code>).
static void on_report(void *ctx, IOReturn result, void *sender, IOHIDReportType type,
                      uint32_t id, uint8_t *report, CFIndex len) {
	if (have || id != 0x11 || len < REPORT_LEN) return;
	int echo = report[2] == want[0] && report[3] == want[1];
	int err = report[2] == 0xff && report[3] == want[0] && report[4] == want[1];
	if (!echo && !err) return;
	memcpy(reply, report, REPORT_LEN);
	have = 1;
}

// Matches the HID interface of a Beam LX that carries the Logitech vendor
// collection, in case the light exposes more than one.
static CFDictionaryRef matching(int pid) {
	int vid = 0x046d, page = 0xff43;
	CFNumberRef v = CFNumberCreate(NULL, kCFNumberIntType, &vid);
	CFNumberRef p = CFNumberCreate(NULL, kCFNumberIntType, &pid);
	CFNumberRef u = CFNumberCreate(NULL, kCFNumberIntType, &page);
	const void *keys[] = {CFSTR(kIOHIDVendorIDKey), CFSTR(kIOHIDProductIDKey), CFSTR(kIOHIDDeviceUsagePageKey)};
	const void *vals[] = {v, p, u};
	CFDictionaryRef match = CFDictionaryCreate(NULL, keys, vals, 3,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFRelease(v);
	CFRelease(p);
	CFRelease(u);
	return match;
}

static int product_id(IOHIDDeviceRef d) {
	int pid = 0;
	CFTypeRef p = IOHIDDeviceGetProperty(d, CFSTR(kIOHIDProductIDKey));
	if (p) CFNumberGetValue(p, kCFNumberIntType, &pid);
	return pid;
}

// Opens the Litra Beam LX connected over USB or Bluetooth, without seizing it.
// USB wins when the light is reachable both ways.
static int litra_open(void) {
	const void *dicts[] = {matching(PID_USB), matching(PID_BLE)};
	CFArrayRef matches = CFArrayCreate(NULL, dicts, 2, &kCFTypeArrayCallBacks);
	mgr = IOHIDManagerCreate(kCFAllocatorDefault, kIOHIDOptionsTypeNone);
	IOHIDManagerSetDeviceMatchingMultiple(mgr, matches);
	CFRelease(matches);
	CFRelease(dicts[0]);
	CFRelease(dicts[1]);

	CFSetRef set = IOHIDManagerCopyDevices(mgr);
	CFIndex n = set ? CFSetGetCount(set) : 0;
	if (n == 0) {
		if (set) CFRelease(set);
		return NOT_FOUND;
	}
	const void **devs = malloc(n * sizeof(*devs));
	CFSetGetValues(set, devs);
	dev = (IOHIDDeviceRef)devs[0];
	for (CFIndex i = 1; i < n; i++)
		if (product_id((IOHIDDeviceRef)devs[i]) == PID_USB) dev = (IOHIDDeviceRef)devs[i];
	CFRetain(dev);
	free(devs);
	CFRelease(set);

	IOReturn r = IOHIDDeviceOpen(dev, kIOHIDOptionsTypeNone);
	if (r != kIOReturnSuccess) return (int)r;
	IOHIDDeviceRegisterInputReportCallback(dev, inbuf, sizeof(inbuf), on_report, NULL);
	IOHIDDeviceScheduleWithRunLoop(dev, CFRunLoopGetCurrent(), kCFRunLoopDefaultMode);
	return 0;
}

// Sends one report and pumps the run loop until its reply arrives.
static int litra_request(const uint8_t *out, uint8_t *in, double timeout) {
	want[0] = out[2];
	want[1] = out[3];
	have = 0;
	// A signal delivered to this thread aborts the kernel's wait for the
	// Bluetooth write and the call fails with kIOReturnError; the Go runtime
	// sends SIGURG to preempt threads, so block signals for the duration.
	sigset_t all, old;
	sigfillset(&all);
	pthread_sigmask(SIG_SETMASK, &all, &old);
	IOReturn r = IOHIDDeviceSetReport(dev, kIOHIDReportTypeOutput, out[0], out, REPORT_LEN);
	pthread_sigmask(SIG_SETMASK, &old, NULL);
	if (r != kIOReturnSuccess) return (int)r;
	CFAbsoluteTime deadline = CFAbsoluteTimeGetCurrent() + timeout;
	while (!have && CFAbsoluteTimeGetCurrent() < deadline)
		CFRunLoopRunInMode(kCFRunLoopDefaultMode, 0.05, true);
	if (!have) return TIMEOUT;
	memcpy(in, reply, REPORT_LEN);
	return 0;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

var opened bool

// request sends one 20-byte report and returns the light's reply to it,
// opening the device on first use.
func request(out []byte) ([]byte, error) {
	if !opened {
		switch rc := C.litra_open(); rc {
		case 0:
		case C.NOT_FOUND:
			return nil, errors.New("Litra Beam LX not found (is it connected via USB or Bluetooth?)")
		default:
			return nil, fmt.Errorf("cannot open the light: IOReturn 0x%08x", uint32(rc))
		}
		opened = true
	}
	in := make([]byte, reportLen)
	switch rc := C.litra_request((*C.uint8_t)(unsafe.Pointer(&out[0])), (*C.uint8_t)(unsafe.Pointer(&in[0])), 1.0); rc {
	case 0:
		return in, nil
	case C.TIMEOUT:
		return nil, errors.New("no reply from the light")
	default:
		return nil, fmt.Errorf("cannot send to the light: IOReturn 0x%08x", uint32(rc))
	}
}
