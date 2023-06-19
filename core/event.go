package core

import (
	"sync"

	"github.com/cometbft/cometbft/libs/events"
)

var once sync.Once
var eventInstance events.EventSwitch

func EventInstance() events.EventSwitch {
	once.Do(func() {
		eventInstance = events.NewEventSwitch()
	})
	return eventInstance
}
