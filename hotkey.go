//go:build windows

package main

// 快捷键只在放映期间注册：RegisterHotKey 是全局的，常驻注册会抢走
// 浏览器刷新(Ctrl+R)、WPS/Office 居中(Ctrl+E)等常用组合键。
var hotkeys = []struct {
	id   uintptr
	vk   uintptr
	name string
}{
	{hotkeyToggle, 'R', "Ctrl+R 计时开关"},
	{hotkeyExtend, 'E', "Ctrl+E 加 5 分钟"},
}

func (a *timerApp) registerHotkeys() {
	for _, hk := range hotkeys {
		if r, _, err := procRegisterHotKey.Call(a.hwnd, hk.id, modControl|modNoRepeat, hk.vk); r == 0 {
			a.debug.logf("注册快捷键失败（可能被其他程序占用）%s: %v", hk.name, err)
		}
	}
}

func (a *timerApp) unregisterHotkeys() {
	for _, hk := range hotkeys {
		procUnregisterHotKey.Call(a.hwnd, hk.id)
	}
}

func (a *timerApp) onHotkey(id uintptr) {
	switch id {
	case hotkeyToggle:
		if a.timerOn {
			a.stopTimer()
		} else {
			a.startTimer()
		}
	case hotkeyExtend:
		if a.timerOn {
			a.roundEnd = a.roundEnd.Add(extendBy)
			a.debug.logf("本轮加时 %v", extendBy)
		}
	}
	a.refresh()
}
