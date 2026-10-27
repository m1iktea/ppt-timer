package main

import (
	"fmt"
	"math"
	"time"
)

// extendBy 是快捷键「加时」每次延长的时长。
const extendBy = 5 * time.Minute

// advanceRound 本轮到点后推进到下一轮截止时间（可能一次跨多轮，如电脑休眠唤醒）。
// 未到点则原样返回，因此加时只影响当前这一轮。
func advanceRound(roundEnd time.Time, period time.Duration, now time.Time) time.Time {
	if period <= 0 || now.Before(roundEnd) {
		return roundEnd
	}
	rounds := now.Sub(roundEnd)/period + 1
	return roundEnd.Add(rounds * period)
}

// remainingText 返回距本轮截止的剩余时间 MM:SS，不足一秒向上取整。
func remainingText(roundEnd, now time.Time) string {
	secs := int(math.Ceil(roundEnd.Sub(now).Seconds()))
	if secs < 0 {
		secs = 0
	}
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}
