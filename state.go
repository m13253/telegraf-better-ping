package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/m13253/telegraf-better-ping/csprng"
	"github.com/m13253/telegraf-better-ping/params"
)

type appState struct {
	Params       *params.PingParams
	Destinations []destinationState
	epoch        time.Time
	lastNow      atomic.Int64
	rng          csprng.CSPRNG
}

type destinationState struct {
	Params *params.DestinationParams
	ID     uint16
	Cipher [2]atomic.Value
}

func NewApp(params *params.PingParams) (app *appState, err error) {
	app = &appState{
		Params:       params,
		Destinations: make([]destinationState, 0, len(params.Destinations)),
		epoch:        time.Now(),
	}
	for i := range params.Destinations {
		app.Destinations = append(app.Destinations, destinationState{
			Params: &params.Destinations[i],
		})
		dest := &app.Destinations[i]
		dest.ID, err = app.rng.UInt16()
		if err != nil {
			err = fmt.Errorf("failed to initialize destination %s: %w", params.Destinations[i].Destination, err)
			return
		}
	}
	return
}

// Return a time.Time with a Unix timestamp strictly later than the previous call.
// This is to ensure leap seconds will not overwrite earlier data into InfluxDB.
func (app *appState) nextUnixTime(now time.Time) time.Time {
	sinceEpoch := now.Round(0).Sub(app.epoch).Nanoseconds()
	for {
		lastSinceEpoch := app.lastNow.Load()
		diff := lastSinceEpoch - sinceEpoch
		if diff >= 0 {
			now = now.Add(time.Duration(diff+1) * time.Nanosecond)
			sinceEpoch += diff + 1
		}
		if app.lastNow.CompareAndSwap(lastSinceEpoch, sinceEpoch) {
			return now
		}
	}
}
