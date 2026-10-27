//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Win32 常量（仅用到的部分）
const (
	wmDestroy    = 0x0002
	wmPaint      = 0x000F
	wmClose      = 0x0010
	wmEraseBkgnd = 0x0014
	wmTimer      = 0x0113
	wmHotkey     = 0x0312

	modControl  = 0x0002
	modNoRepeat = 0x4000

	hotkeyToggle = 1
	hotkeyExtend = 2

	wsPopup   = 0x80000000
	wsCaption = 0x00C00000

	wsExTopmost     = 0x00000008
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080
	wsExLayered     = 0x00080000
	wsExNoActivate  = 0x08000000

	swHide = 0

	swpNoActivate = 0x0010
	swpShowWindow = 0x0040

	lwaColorKey = 0x1

	antialiasedQuality = 4 // 灰度抗锯齿；ClearType 彩边在色键透明下会发虚

	dtCenter     = 0x1
	dtVCenter    = 0x4
	dtSingleLine = 0x20

	bkTransparent = 1
	fwBold        = 700
	logPixelsY    = 90

	spiGetWorkArea       = 0x0030
	monitorDefaultToNear = 2

	mbIconError = 0x10

	processQueryLimitedInformation = 0x1000
)

var (
	hwndTopmost       = ^uintptr(0) // (HWND)-1
	gwlStyle    int32 = -16         // 变量而非常量，转 uintptr 时按符号扩展
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW           = user32.NewProc("RegisterClassExW")
	procCreateWindowExW            = user32.NewProc("CreateWindowExW")
	procDefWindowProcW             = user32.NewProc("DefWindowProcW")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procPostQuitMessage            = user32.NewProc("PostQuitMessage")
	procPostMessageW               = user32.NewProc("PostMessageW")
	procFindWindowW                = user32.NewProc("FindWindowW")
	procSetTimer                   = user32.NewProc("SetTimer")
	procShowWindow                 = user32.NewProc("ShowWindow")
	procSetWindowPos               = user32.NewProc("SetWindowPos")
	procInvalidateRect             = user32.NewProc("InvalidateRect")
	procBeginPaint                 = user32.NewProc("BeginPaint")
	procEndPaint                   = user32.NewProc("EndPaint")
	procGetClientRect              = user32.NewProc("GetClientRect")
	procGetWindowRect              = user32.NewProc("GetWindowRect")
	procGetWindowLongW             = user32.NewProc("GetWindowLongW")
	procIsZoomed                   = user32.NewProc("IsZoomed")
	procFillRect                   = user32.NewProc("FillRect")
	procDrawTextW                  = user32.NewProc("DrawTextW")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procMonitorFromWindow          = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW            = user32.NewProc("GetMonitorInfoW")
	procSystemParametersInfoW      = user32.NewProc("SystemParametersInfoW")
	procSetProcessDPIAware         = user32.NewProc("SetProcessDPIAware")
	procGetDC                      = user32.NewProc("GetDC")
	procReleaseDC                  = user32.NewProc("ReleaseDC")
	procMessageBoxW                = user32.NewProc("MessageBoxW")
	procRegisterHotKey             = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey           = user32.NewProc("UnregisterHotKey")

	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procGetDeviceCaps    = gdi32.NewProc("GetDeviceCaps")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

type rect struct {
	Left, Top, Right, Bottom int32
}

type point struct {
	X, Y int32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

type paintStruct struct {
	Hdc       uintptr
	Erase     int32
	RcPaint   rect
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

func utf16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func rgb(r, g, b byte) uintptr {
	return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16
}

func messageBox(text string) {
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(utf16(text))),
		uintptr(unsafe.Pointer(utf16(appTitle))), mbIconError)
}

func windowRect(hwnd uintptr) rect {
	var r rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return r
}

func monitorRectOf(hwnd uintptr) rect {
	hmon, _, _ := procMonitorFromWindow.Call(hwnd, monitorDefaultToNear)
	mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&mi)))
	return mi.Monitor
}

// primaryWorkArea 返回主屏去掉任务栏后的可用区域。
func primaryWorkArea() rect {
	var r rect
	procSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0)
	return r
}

func screenDPI() int32 {
	hdc, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, hdc)
	dpi, _, _ := procGetDeviceCaps.Call(hdc, logPixelsY)
	if dpi == 0 {
		return 96
	}
	return int32(dpi)
}
