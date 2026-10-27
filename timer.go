//go:build windows

package main

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

const (
	windowClass = "PPTCountdownTimer"
	tickID      = 1
	tickMillis  = 500
	// 连续这么多次检测不到放映才判定退出，避免 WPS 切换动画时窗口短暂消失导致误关
	stopAfterMisses = 3

	baseWidth    = 180 // 96 DPI 下的尺寸，运行时按 DPI 缩放
	baseHeight   = 70
	baseFontSize = 48
	baseMargin   = 16
	// 透明底：用色键把背景色整块抠掉，只留下文字
	transparentKey = 0xFFFFFF // 白色，与 rgb(255,255,255) 一致
)

type timerApp struct {
	cfg       config
	cfgPath   string
	hwnd      uintptr
	font      uintptr
	bgBrush   uintptr
	dpi       int32
	running   bool
	startedAt time.Time
	period    time.Duration
	misses    int
	debug     *debugLog
}

var app *timerApp

func scale(v, dpi int32) int32 { return v * dpi / 96 }

func (a *timerApp) createWindow() error {
	inst, _, _ := procGetModuleHandleW.Call(0)
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   syscall.NewCallback(wndProc),
		Instance:  inst,
		ClassName: utf16(windowClass),
	}
	if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("注册窗口类失败: %w", err)
	}
	exStyle := uintptr(wsExTopmost | wsExToolWindow | wsExLayered | wsExTransparent | wsExNoActivate)
	hwnd, _, err := procCreateWindowExW.Call(exStyle,
		uintptr(unsafe.Pointer(utf16(windowClass))), uintptr(unsafe.Pointer(utf16(appTitle))),
		wsPopup, 0, 0, uintptr(scale(baseWidth, a.dpi)), uintptr(scale(baseHeight, a.dpi)),
		0, 0, inst, 0)
	if hwnd == 0 {
		return fmt.Errorf("创建窗口失败: %w", err)
	}
	a.hwnd = hwnd
	procSetLayeredWindowAttributes.Call(hwnd, transparentKey, 0, lwaColorKey)
	a.font, _, _ = procCreateFontW.Call(uintptr(-scale(baseFontSize, a.dpi)), 0, 0, 0, fwBold,
		0, 0, 0, 0, 0, 0, antialiasedQuality, 0, uintptr(unsafe.Pointer(utf16("Segoe UI"))))
	a.bgBrush, _, _ = procCreateSolidBrush.Call(transparentKey)
	procSetTimer.Call(hwnd, tickID, tickMillis, 0)
	return nil
}

func (a *timerApp) onTick() {
	active := slideshowActive()
	a.debug.maybeDump(active)
	switch {
	case active:
		a.misses = 0
		if !a.running {
			a.start()
		}
	case a.running:
		a.misses++
		if a.misses >= stopAfterMisses {
			a.stop()
		}
	}
	if a.running {
		a.place() // 每次都重新置顶，防止被全屏放映窗口压住
		procInvalidateRect.Call(a.hwnd, 0, 0)
	}
}

func (a *timerApp) start() {
	a.reloadConfig()
	a.running = true
	a.startedAt = time.Now()
	a.period = time.Duration(a.cfg.Minutes * float64(time.Minute))
	a.debug.logf("放映开始，每轮倒计时 %.1f 分钟，到点自动进入下一轮", a.cfg.Minutes)
}

func (a *timerApp) stop() {
	a.running = false
	a.misses = 0
	procShowWindow.Call(a.hwnd, swHide)
	a.debug.logf("放映结束，计时窗关闭")
}

// place 把计时窗放到主屏工作区的指定角落（扩展模式下主屏即讲者自己的屏幕）。
func (a *timerApp) place() {
	work := primaryWorkArea()
	w, h, m := scale(baseWidth, a.dpi), scale(baseHeight, a.dpi), scale(baseMargin, a.dpi)
	x, y := work.Right-w-m, work.Top+m
	switch a.cfg.Position {
	case "top-left":
		x = work.Left + m
	case "bottom-right":
		y = work.Bottom - h - m
	case "bottom-left":
		x, y = work.Left+m, work.Bottom-h-m
	}
	procSetWindowPos.Call(a.hwnd, hwndTopmost, uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		swpNoActivate|swpShowWindow)
}

func (a *timerApp) paint() {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(a.hwnd, uintptr(unsafe.Pointer(&ps)))
	defer procEndPaint.Call(a.hwnd, uintptr(unsafe.Pointer(&ps)))

	var rc rect
	procGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&rc)))
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), a.bgBrush)
	old, _, _ := procSelectObject.Call(hdc, a.font)
	defer procSelectObject.Call(hdc, old)
	procSetTextColor.Call(hdc, rgb(0, 0, 0))
	procSetBkMode.Call(hdc, bkTransparent)

	text, _ := syscall.UTF16FromString(remainingText(a.startedAt, a.period, time.Now()))
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&text[0])), uintptr(len(text)-1),
		uintptr(unsafe.Pointer(&rc)), dtCenter|dtVCenter|dtSingleLine)
}

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmTimer:
		app.onTick()
		return 0
	case wmPaint:
		app.paint()
		return 0
	case wmEraseBkgnd:
		return 1
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

func runMessageLoop() {
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// reloadConfig 每次开始放映时重读配置，改完 ini 不用重启程序；配置有误则沿用上次的。
func (a *timerApp) reloadConfig() {
	cfg, err := loadConfig(a.cfgPath)
	if err != nil {
		a.debug.logf("配置有误，沿用上次配置: %v", err)
		return
	}
	a.cfg = cfg
}
