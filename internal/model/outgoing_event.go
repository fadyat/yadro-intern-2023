package model

import (
	"fmt"
	"time"
)

type OutgoingEvent struct {

	// HappensAt is the time when the event happens.
	//
	// Not equal to the time when the event was received, for example:
	//  if user was in the queue, happens time equals to the time event was popped
	HappensAt time.Time

	Type OutgoingEventType

	// Err is an error that occurred while processing the event.
	// If the event was successful, then the error will be nil.
	Err error

	// ClientData is the data of the client who caused the event.
	// If the event was successful, then the data will be shown in the output.
	Client ClientData
}

func (e *OutgoingEvent) String(timeFormat string) string {
	if e.Err != nil {
		return fmt.Sprintf(
			"%s %d %s",
			e.HappensAt.Format(timeFormat),
			e.Type,
			e.Err.Error(),
		)
	}

	return fmt.Sprintf(
		"%s %d %s",
		e.HappensAt.Format(timeFormat),
		e.Type,
		e.Client.String(),
	)
}

func NewErrorEvent(happensAt time.Time, err error) *OutgoingEvent {
	return &OutgoingEvent{
		HappensAt: happensAt,
		Type:      OutgoingEventTypeError,
		Err:       err,
	}
}

func NewClientLeftEvent(happensAt time.Time, client ClientData) *OutgoingEvent {
	return &OutgoingEvent{
		HappensAt: happensAt,
		Type:      OutgoingEventTypeClientLeft,
		Client:    client,
	}
}

func NewClientSatEvent(happensAt time.Time, client ClientData) *OutgoingEvent {
	return &OutgoingEvent{
		HappensAt: happensAt,
		Type:      OutgoingEventTypeClientSat,
		Client:    client,
	}
}

type OutgoingEventType int

const (
	OutgoingEventTypeClientLeft OutgoingEventType = 11
	OutgoingEventTypeClientSat  OutgoingEventType = 12
	OutgoingEventTypeError      OutgoingEventType = 13
)
