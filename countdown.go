package main

import (
	"fmt"
	"math"
	"time"
)

// remainingText 返回本轮剩余时间 MM:SS。每轮 period，到点自动进入下一轮重新倒数。
func remainingText(startedAt time.Time, period time.Duration, now time.Time) string {
	elapsed := now.Sub(startedAt)
	if elapsed < 0 || period <= 0 {
		elapsed = 0
	}
	left := period
	if period > 0 {
		left = period - elapsed%period
	}
	secs := int(math.Ceil(left.Seconds()))
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}
