//go:build windows

package main

import (
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// 放映程序进程：PowerPoint、WPS 演示（独立模式 wpp.exe / 整合模式 wps.exe）
var presenterProcs = map[string]bool{
	"powerpnt.exe": true,
	"wpp.exe":      true,
	"wps.exe":      true,
}

// 已知的放映窗口类名；未命中时再用"无边框全屏"判断（WPS 走这条）
var slideshowClasses = map[string]bool{
	"screenClass": true, // PowerPoint 放映窗口
}

const maxProcCache = 512

var (
	procNameCache = map[uint32]string{}
	enumFound     bool
	enumCallback  = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		if isSlideshowWindow(hwnd) {
			enumFound = true
			return 0 // 停止枚举
		}
		return 1
	})
)

// slideshowActive 判断当前是否有 PPT / WPS 放映窗口。
func slideshowActive() bool {
	enumFound = false
	_ = windows.EnumWindows(enumCallback, nil)
	return enumFound
}

func isSlideshowWindow(hwnd uintptr) bool {
	h := windows.HWND(hwnd)
	if !windows.IsWindowVisible(h) || !presenterProcs[processName(h)] {
		return false
	}
	return slideshowClasses[className(h)] || isBorderlessFullscreen(hwnd)
}

// isBorderlessFullscreen：无标题栏、非最大化、且铺满整个显示器（含任务栏区域）。
func isBorderlessFullscreen(hwnd uintptr) bool {
	style, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlStyle))
	if uint32(style)&wsCaption == wsCaption {
		return false
	}
	if zoomed, _, _ := procIsZoomed.Call(hwnd); zoomed != 0 {
		return false
	}
	return windowRect(hwnd) == monitorRectOf(hwnd)
}

func className(h windows.HWND) string {
	buf := make([]uint16, 256)
	n, err := windows.GetClassName(h, &buf[0], int32(len(buf)))
	if err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

func processName(h windows.HWND) string {
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(h, &pid); err != nil || pid == 0 {
		return ""
	}
	if name, ok := procNameCache[pid]; ok {
		return name
	}
	name := queryProcessName(pid)
	if len(procNameCache) >= maxProcCache {
		procNameCache = map[uint32]string{}
	}
	procNameCache[pid] = name
	return name
}

func queryProcessName(pid uint32) string {
	hp, err := windows.OpenProcess(processQueryLimitedInformation, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(hp)
	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(hp, 0, &buf[0], &size); err != nil {
		return ""
	}
	return strings.ToLower(filepath.Base(windows.UTF16ToString(buf[:size])))
}
