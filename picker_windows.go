package main

import (
	"fmt"
	"unsafe"
)

var (
	comdlg32DLL = systemDLL("comdlg32.dll")

	procChooseColor          = comdlg32DLL.NewProc("ChooseColorW")
	procCommDlgExtendedError = comdlg32DLL.NewProc("CommDlgExtendedError")
)

const (
	ccRGBInit  = 0x1 // start from the colour in Result
	ccFullOpen = 0x2 // show the full palette, not only the basic colours
)

// CHOOSECOLORW
type chooseColor struct {
	StructSize   uint32
	Owner        uintptr
	Instance     uintptr
	Result       uint32 // COLORREF: 0x00BBGGRR
	CustomColors *[16]uint32
	Flags        uint32
	CustData     uintptr
	Hook         uintptr
	TemplateName *uint16
}

// pickColor shows the Windows colour dialog, starting from white; ok is false
// when it was cancelled.
func pickColor() (r, g, b byte, ok bool, err error) {
	var custom [16]uint32
	cc := chooseColor{Result: 0x00ffffff, CustomColors: &custom, Flags: ccRGBInit | ccFullOpen}
	cc.StructSize = uint32(unsafe.Sizeof(cc))
	if chosen, _, _ := procChooseColor.Call(uintptr(unsafe.Pointer(&cc))); int32(chosen) == 0 {
		// Zero means either Cancel or a failure; only a failure sets an error code.
		if code, _, _ := procCommDlgExtendedError.Call(); uint32(code) != 0 {
			return 0, 0, 0, false, fmt.Errorf("the colour dialog failed (error 0x%04x)", uint32(code))
		}
		return 0, 0, 0, false, nil
	}
	return byte(cc.Result), byte(cc.Result >> 8), byte(cc.Result >> 16), true, nil
}
