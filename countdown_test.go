package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRemainingText(t *testing.T) {
	start := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	five := 5 * time.Minute
	tests := []struct {
		name    string
		period  time.Duration
		elapsed time.Duration
		want    string
	}{
		{"刚开始", five, 0, "05:00"},
		{"不足一秒向上取整", five, 900 * time.Millisecond, "05:00"},
		{"过一秒", five, time.Second, "04:59"},
		{"本轮最后一秒", five, 4*time.Minute + 59*time.Second, "00:01"},
		{"到点进入第二轮", five, five, "05:00"},
		{"第二轮中途", five, five + 90*time.Second, "03:30"},
		{"第三轮末尾", five, 3*five - time.Second, "00:01"},
		{"小数分钟", 150 * time.Second, 30 * time.Second, "02:00"},
		{"超过一小时", 75 * time.Minute, 0, "75:00"},
		{"时钟回拨按刚开始算", five, -10 * time.Second, "05:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := remainingText(start, tt.period, start.Add(tt.elapsed)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    config
		wantErr bool
	}{
		{"默认模板", defaultConfigText, config{5, "top-right"}, false},
		{"小数与 BOM", "\ufeffminutes = 2.5\nposition=Bottom-Left\n", config{2.5, "bottom-left"}, false},
		{"注释与空行", "# x\n\n; y\nminutes=10\n", config{10, "top-right"}, false},
		{"分钟为 0", "minutes=0\n", config{}, true},
		{"分钟非数字", "minutes=abc\n", config{}, true},
		{"位置无效", "position=center\n", config{}, true},
		{"未知配置项", "color=red\n", config{}, true},
		{"缺少等号", "minutes 5\n", config{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), configName)
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := loadConfig(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoadConfigCreatesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), configName)
	got, err := loadConfig(path)
	if err != nil || got != (config{defaultMinutes, defaultPosition}) {
		t.Fatalf("got %+v, %v", got, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != defaultConfigText {
		t.Fatalf("默认配置未写入: %v", err)
	}
}
