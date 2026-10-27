package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultMinutes  = 5
	defaultPosition = "top-right"
	maxMinutes      = 600
	configName      = "ppt-timer.ini"
)

const defaultConfigText = `; PPT 倒计时配置，改完后下次开始放映生效
; minutes: 倒计时分钟数，可以写小数，如 2.5
minutes=5
; position: 计时窗在主屏的位置 top-right / top-left / bottom-right / bottom-left
position=top-right
`

var validPositions = map[string]bool{
	"top-right": true, "top-left": true, "bottom-right": true, "bottom-left": true,
}

type config struct {
	Minutes  float64
	Position string
}

// loadConfig 读取配置；文件不存在时写入默认配置并返回默认值。
func loadConfig(path string) (config, error) {
	cfg := config{Minutes: defaultMinutes, Position: defaultPosition}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		_ = os.WriteFile(path, []byte(defaultConfigText), 0o644)
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("读取配置失败: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		next, err := applyLine(cfg, strings.TrimSpace(scanner.Text()))
		if err != nil {
			return cfg, fmt.Errorf("%s 第 %d 行: %w", path, lineNo, err)
		}
		cfg = next
	}
	return cfg, scanner.Err()
}

func applyLine(cfg config, line string) (config, error) {
	line = strings.TrimPrefix(line, "\ufeff") // 记事本保存的 UTF-8 BOM
	if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
		return cfg, nil
	}
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return cfg, fmt.Errorf("格式应为 key=value")
	}
	key, value = strings.ToLower(strings.TrimSpace(key)), strings.TrimSpace(value)
	switch key {
	case "minutes":
		m, err := strconv.ParseFloat(value, 64)
		if err != nil || m <= 0 || m > maxMinutes {
			return cfg, fmt.Errorf("minutes 必须是 0~%d 之间的数字，当前为 %q", maxMinutes, value)
		}
		cfg.Minutes = m
	case "position":
		p := strings.ToLower(value)
		if !validPositions[p] {
			return cfg, fmt.Errorf("position 取值无效: %q", value)
		}
		cfg.Position = p
	default:
		return cfg, fmt.Errorf("未知配置项 %q", key)
	}
	return cfg, nil
}
