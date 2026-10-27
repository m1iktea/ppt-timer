//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const dumpEvery = 4 * time.Second

// debugLog 在 --debug 模式下把放映相关窗口信息写到日志，用于在新机器上核对检测逻辑。
// 为 nil 时所有方法都是空操作。
type debugLog struct {
	f        *os.File
	lastDump time.Time
}

func openDebugLog(path string) (*debugLog, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("打开调试日志失败: %w", err)
	}
	return &debugLog{f: f}, nil
}

func (d *debugLog) logf(format string, args ...any) {
	if d == nil {
		return
	}
	fmt.Fprintf(d.f, "%s %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

// maybeDump 定期列出 PPT/WPS 进程的所有可见窗口及判定结果。
func (d *debugLog) maybeDump(active bool) {
	if d == nil || time.Since(d.lastDump) < dumpEvery {
		return
	}
	d.lastDump = time.Now()
	d.logf("---- 检测结果 active=%v ----", active)
	dumpTarget = d
	_ = windows.EnumWindows(dumpCallback, nil)
}

// syscall.NewCallback 数量有上限且不回收，只能创建一次复用。
var (
	dumpTarget   *debugLog
	dumpCallback = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		dumpTarget.dumpWindow(hwnd)
		return 1
	})
)

func (d *debugLog) dumpWindow(hwnd uintptr) {
	h := windows.HWND(hwnd)
	proc := processName(h)
	if !windows.IsWindowVisible(h) || !presenterProcs[proc] {
		return
	}
	style, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlStyle))
	zoomed, _, _ := procIsZoomed.Call(hwnd)
	d.logf("  proc=%s class=%q rect=%v monitor=%v style=0x%08X zoomed=%v match=%v",
		proc, className(h), windowRect(hwnd), monitorRectOf(hwnd),
		uint32(style), zoomed != 0, isSlideshowWindow(hwnd))
}
