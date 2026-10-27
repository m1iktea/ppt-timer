package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRemainingText(t *testing.T) {
	now := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		left time.Duration
		want string
	}{
		{"整 5 分钟", 5 * time.Minute, "05:00"},
		{"不足一秒向上取整", 4*time.Minute + 59*time.Second + 100*time.Millisecond, "05:00"},
		{"过一秒", 4*time.Minute + 59*time.Second, "04:59"},
		{"加时后超过一轮", 9*time.Minute + 30*time.Second, "09:30"},
		{"超过一百分钟", 125 * time.Minute, "125:00"},
		{"已过截止", -time.Second, "00:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := remainingText(now.Add(tt.left), now); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAdvanceRound(t *testing.T) {
	end := time.Date(2026, 10, 27, 10, 5, 0, 0, time.UTC)
	five := 5 * time.Minute
	tests := []struct {
		name   string
		period time.Duration
		now    time.Duration // 相对 end
		want   time.Duration // 相对 end
	}{
		{"本轮未到点不变", five, -time.Second, 0},
		{"刚好到点进入下一轮", five, 0, five},
		{"下一轮中途", five, 90 * time.Second, five},
		{"跨过多轮（如休眠唤醒）", five, 12 * time.Minute, 3 * five},
		{"周期非法不推进", 0, time.Minute, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := advanceRound(end, tt.period, end.Add(tt.now))
			if want := end.Add(tt.want); !got.Equal(want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestLoopAndExtend(t *testing.T) {
	start := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	five := 5 * time.Minute
	end := start.Add(five)
	at := func(d time.Duration) string {
		now := start.Add(d)
		end = advanceRound(end, five, now)
		return remainingText(end, now)
	}
	if got := at(4*time.Minute + 59*time.Second); got != "00:01" {
		t.Fatalf("第一轮末尾 got %q", got)
	}
	if got := at(five); got != "05:00" {
		t.Fatalf("进入第二轮 got %q", got)
	}
	end = end.Add(extendBy) // 第二轮开头加 5 分钟
	if got := at(five + time.Minute); got != "09:00" {
		t.Fatalf("加时后 got %q", got)
	}
	if got := at(3 * five); got != "05:00" {
		t.Fatalf("加时那轮结束后回到正常周期 got %q", got)
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
