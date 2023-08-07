package model

import (
	"fmt"
	"time"
)

type WrappedIncomingEvent struct {
	Event *IncomingEvent
	Err   error
}

type IncomingEvent struct {

	// HappensAt is the time when the event happens.
	//
	// Can be before/after the computer club opens/closes.
	HappensAt time.Time

	Type   IncomingEventType
	Client ClientData
}

func NewIncomingEvent(
	happensAt time.Time, eventType IncomingEventType, client ClientData,
) *IncomingEvent {
	return &IncomingEvent{
		HappensAt: happensAt,
		Type:      eventType,
		Client:    client,
	}
}

func (e *IncomingEvent) String(timeFormat string) string {
	return fmt.Sprintf(
		"%s %d %s",
		e.HappensAt.Format(timeFormat),
		e.Type,
		e.Client.String(),
	)
}

type IncomingEventType int

const (
	Arrives IncomingEventType = 1
	Sits    IncomingEventType = 2
	Waits   IncomingEventType = 3
	Leaves  IncomingEventType = 4
)

func GetValidClientDataSize(eventType IncomingEventType) int {
	switch eventType {
	case Sits:
		return 2
	case Waits, Leaves, Arrives:
		return 1
	}

	return 0
}
