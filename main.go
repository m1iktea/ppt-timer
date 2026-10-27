//go:build windows

// ppt-timer：检测到 PowerPoint / WPS 开始放映时，在主屏角落显示置顶倒计时，放映结束自动隐藏。
package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	appTitle     = "PPT 倒计时"
	mutexName    = "Local\\PPTCountdownTimer"
	debugLogName = "ppt-timer-debug.log"
)

func main() {
	runtime.LockOSThread() // Win32 窗口消息必须在创建窗口的线程上处理

	if hasArg("--quit") {
		quitRunningInstance()
		return
	}
	if !acquireSingleInstance() {
		return // 已在运行
	}
	if err := run(); err != nil {
		messageBox(err.Error())
		os.Exit(1)
	}
}

func run() error {
	procSetProcessDPIAware.Call()
	dir, err := exeDir()
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(dir, configName)
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		return err
	}
	app = &timerApp{cfg: cfg, cfgPath: cfgPath, dpi: screenDPI()}
	if hasArg("--debug") {
		if app.debug, err = openDebugLog(filepath.Join(dir, debugLogName)); err != nil {
			return err
		}
		app.debug.logf("启动 minutes=%.1f position=%s dpi=%d", cfg.Minutes, cfg.Position, app.dpi)
	}
	if err := app.createWindow(); err != nil {
		return err
	}
	runMessageLoop()
	return nil
}

func hasArg(name string) bool {
	for _, a := range os.Args[1:] {
		if a == name {
			return true
		}
	}
	return false
}

func exeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", errors.New("无法定位程序所在目录: " + err.Error())
	}
	return filepath.Dir(exe), nil
}

// acquireSingleInstance 用命名互斥量保证只运行一个实例。句柄故意不关闭，随进程释放。
func acquireSingleInstance() bool {
	_, err := windows.CreateMutex(nil, false, utf16(mutexName))
	return !errors.Is(err, windows.ERROR_ALREADY_EXISTS)
}

func quitRunningInstance() {
	hwnd, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(utf16(windowClass))), 0)
	if hwnd != 0 {
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
	}
}
