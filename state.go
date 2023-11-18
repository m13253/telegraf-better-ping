package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/m13253/telegraf-better-ping/csprng"
	"github.com/m13253/telegraf-better-ping/params"
)

type AppState struct {
	Params          *params.PingParams
	Epoch           time.Time
	LastNow         atomic.Int64
	RandomGenerator csprng.CSPRNG
	Destinations    []DestinationState
}

type DestinationState struct {
	Params *params.DestinationParams
	ID     uint16
	Crypt  [2]atomic.Value
}

func NewApp(params *params.PingParams) (state *AppState, err error) {
	state = &AppState{
		Params:       params,
		Epoch:        time.Now(),
		Destinations: make([]DestinationState, 0, len(params.Destinations)),
	}
	for i := range params.Destinations {
		state.Destinations = append(state.Destinations, DestinationState{
			Params: &params.Destinations[i],
		})
		dest := &state.Destinations[i]
		dest.ID, err = state.RandomGenerator.UInt16()
		if err != nil {
			err = fmt.Errorf("failed to initialize destination %s: %w", params.Destinations[i].Destination, err)
			return
		}
	}
	return
}
