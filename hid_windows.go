package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	vendorID   = 0x046d
	productUSB = 0xc903
	productBLE = 0xb903
	usagePage  = 0xff43 // the Logitech vendor collection that carries HID++

	digcfPresent         = 0x02
	digcfDeviceInterface = 0x10
	hidpStatusSuccess    = 0x00110000
)

// systemDLL loads a DLL from System32, never from the program's own folder.
func systemDLL(name string) *syscall.LazyDLL {
	if root := os.Getenv("SystemRoot"); root != "" {
		name = filepath.Join(root, "System32", name)
	}
	return syscall.NewLazyDLL(name)
}

var (
	hidDLL      = systemDLL("hid.dll")
	setupapiDLL = systemDLL("setupapi.dll")

	procGetHidGuid        = hidDLL.NewProc("HidD_GetHidGuid")
	procGetAttributes     = hidDLL.NewProc("HidD_GetAttributes")
	procGetPreparsedData  = hidDLL.NewProc("HidD_GetPreparsedData")
	procFreePreparsedData = hidDLL.NewProc("HidD_FreePreparsedData")
	procGetCaps           = hidDLL.NewProc("HidP_GetCaps")

	procGetClassDevs             = setupapiDLL.NewProc("SetupDiGetClassDevsW")
	procEnumDeviceInterfaces     = setupapiDLL.NewProc("SetupDiEnumDeviceInterfaces")
	procGetDeviceInterfaceDetail = setupapiDLL.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procDestroyDeviceInfoList    = setupapiDLL.NewProc("SetupDiDestroyDeviceInfoList")
)

// SP_DEVICE_INTERFACE_DATA
type deviceInterfaceData struct {
	Size      uint32
	ClassGUID syscall.GUID
	Flags     uint32
	Reserved  uintptr
}

// HIDD_ATTRIBUTES
type hiddAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

// HIDP_CAPS; only the leading fields are used.
type hidpCaps struct {
	Usage                   uint16
	UsagePage               uint16
	InputReportByteLength   uint16
	OutputReportByteLength  uint16
	FeatureReportByteLength uint16
	Rest                    [27]uint16
}

// Set BEAMCTL_DEBUG=1 to see which HID devices were found and why one was chosen.
var debug = os.Getenv("BEAMCTL_DEBUG") != ""

func debugf(format string, args ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "debug: "+format+"\n", args...)
	}
}

// hidPaths lists the device paths of all present HID collections. Windows
// exposes each top-level collection of a device as a device of its own.
func hidPaths() ([]string, error) {
	var guid syscall.GUID
	procGetHidGuid.Call(uintptr(unsafe.Pointer(&guid)))
	set, _, err := procGetClassDevs.Call(uintptr(unsafe.Pointer(&guid)), 0, 0, digcfPresent|digcfDeviceInterface)
	if syscall.Handle(set) == syscall.InvalidHandle {
		return nil, fmt.Errorf("cannot list HID devices: %w", err)
	}
	defer procDestroyDeviceInfoList.Call(set)

	// cbSize of SP_DEVICE_INTERFACE_DETAIL_DATA_W: a DWORD and one WCHAR,
	// packed on 32-bit Windows and padded on 64-bit.
	detailSize := uint32(6)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		detailSize = 8
	}

	var paths []string
	for index := uintptr(0); ; index++ {
		var data deviceInterfaceData
		data.Size = uint32(unsafe.Sizeof(data))
		ok, _, _ := procEnumDeviceInterfaces.Call(set, 0, uintptr(unsafe.Pointer(&guid)), index, uintptr(unsafe.Pointer(&data)))
		if int32(ok) == 0 {
			break // no more devices
		}
		// The first call only reports how large the detail buffer has to be.
		var size uint32
		procGetDeviceInterfaceDetail.Call(set, uintptr(unsafe.Pointer(&data)), 0, 0, uintptr(unsafe.Pointer(&size)), 0)
		if size <= 4 {
			continue
		}
		detail := make([]byte, size)
		binary.LittleEndian.PutUint32(detail, detailSize)
		ok, _, _ = procGetDeviceInterfaceDetail.Call(set, uintptr(unsafe.Pointer(&data)), uintptr(unsafe.Pointer(&detail[0])), uintptr(size), 0, 0)
		if int32(ok) == 0 {
			continue
		}
		paths = append(paths, utf16String(detail[4:]))
	}
	return paths, nil
}

// utf16String decodes a NUL-terminated little-endian UTF-16 string.
func utf16String(b []byte) string {
	var units []uint16
	for i := 0; i+1 < len(b); i += 2 {
		unit := binary.LittleEndian.Uint16(b[i:])
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	return string(utf16.Decode(units))
}

func open(path string, access uint32) (syscall.Handle, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	return syscall.CreateFile(p, access, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, 0, 0)
}

type hidInfo struct {
	path                string
	vendor, product     uint16
	usagePage, usage    uint16
	inputLen, outputLen uint16
}

func describe(path string) (hidInfo, error) {
	// Reading a device's identity needs no access rights, so this also works
	// for keyboards and mice, which Windows keeps to itself.
	h, err := open(path, 0)
	if err != nil {
		return hidInfo{}, err
	}
	defer syscall.CloseHandle(h)

	var attrs hiddAttributes
	attrs.Size = uint32(unsafe.Sizeof(attrs))
	if ok, _, err := procGetAttributes.Call(uintptr(h), uintptr(unsafe.Pointer(&attrs))); byte(ok) == 0 {
		return hidInfo{}, fmt.Errorf("HidD_GetAttributes: %w", err)
	}
	var preparsed uintptr
	if ok, _, err := procGetPreparsedData.Call(uintptr(h), uintptr(unsafe.Pointer(&preparsed))); byte(ok) == 0 {
		return hidInfo{}, fmt.Errorf("HidD_GetPreparsedData: %w", err)
	}
	defer procFreePreparsedData.Call(preparsed)
	var caps hidpCaps
	if status, _, _ := procGetCaps.Call(preparsed, uintptr(unsafe.Pointer(&caps))); uint32(status) != hidpStatusSuccess {
		return hidInfo{}, fmt.Errorf("HidP_GetCaps: status 0x%08x", uint32(status))
	}
	return hidInfo{
		path:      path,
		vendor:    attrs.VendorID,
		product:   attrs.ProductID,
		usagePage: caps.UsagePage,
		usage:     caps.Usage,
		inputLen:  caps.InputReportByteLength,
		outputLen: caps.OutputReportByteLength,
	}, nil
}

type light struct {
	write   syscall.Handle
	outLen  int
	reports chan []byte // closed when reading fails; readErr then says why
	readErr error
}

var dev *light

// openLight opens the Litra Beam LX connected over USB or Bluetooth. USB wins
// when the light is reachable both ways.
func openLight() (*light, error) {
	paths, err := hidPaths()
	if err != nil {
		return nil, err
	}
	var found *hidInfo
	for _, path := range paths {
		info, err := describe(path)
		if err != nil {
			debugf("skipped %s: %v", path, err)
			continue
		}
		if info.vendor != vendorID {
			continue
		}
		debugf("Logitech HID: product %04x, usage %04x:%04x, input %d bytes, output %d bytes, %s",
			info.product, info.usagePage, info.usage, info.inputLen, info.outputLen, info.path)
		if info.usagePage != usagePage || (info.product != productUSB && info.product != productBLE) {
			continue
		}
		if info.inputLen < reportLen || info.outputLen < reportLen {
			continue // not the collection that carries the 20-byte reports
		}
		if found == nil || info.product == productUSB {
			found = &info
		}
	}
	if found == nil {
		return nil, errors.New("Litra Beam LX not found (is it connected via USB or Bluetooth?)")
	}
	debugf("using product %04x", found.product)

	// Two handles: these are synchronous, and on a single one the pending
	// read would hold up every write.
	const readWrite = syscall.GENERIC_READ | syscall.GENERIC_WRITE
	write, err := open(found.path, readWrite)
	if err != nil {
		return nil, fmt.Errorf("cannot open the light: %w", err)
	}
	read, err := open(found.path, readWrite)
	if err != nil {
		return nil, fmt.Errorf("cannot open the light: %w", err)
	}
	l := &light{write: write, outLen: int(found.outputLen), reports: make(chan []byte, 64)}
	go l.readReports(read, int(found.inputLen))
	return l, nil
}

func (l *light) readReports(h syscall.Handle, size int) {
	for {
		buf := make([]byte, size)
		var n uint32
		if err := syscall.ReadFile(h, buf, &n, nil); err != nil {
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

const (
	writeTimeout = 2 * time.Second
	replyTimeout = time.Second
)

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

	// Windows wants the buffer to be exactly as long as the device's output report.
	buf := make([]byte, dev.outLen)
	copy(buf, out)
	// A write to a light that is paired but out of reach may never return, so
	// it runs on the side and gets a deadline of its own.
	written := make(chan error, 1)
	go func() {
		var n uint32
		written <- syscall.WriteFile(dev.write, buf, &n, nil)
	}()
	select {
	case err := <-written:
		if err != nil {
			return nil, fmt.Errorf("cannot send to the light: %w", err)
		}
	case <-time.After(writeTimeout):
		return nil, errors.New("cannot send to the light: timed out")
	}

	timeout := time.After(replyTimeout)
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
