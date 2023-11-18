package main

import "time"

func IncreasingNow(state *AppState) time.Time {
	now := time.Now()
	sinceEpoch := now.Sub(state.Epoch).Nanoseconds()
	for {
		lastSinceEpoch := state.LastNow.Load()
		diff := lastSinceEpoch - sinceEpoch
		if diff >= 0 {
			now = now.Add(time.Duration(diff+1) * time.Nanosecond)
			sinceEpoch += diff + 1
		} else if state.LastNow.CompareAndSwap(lastSinceEpoch, sinceEpoch) {
			return now
		}
	}
}
