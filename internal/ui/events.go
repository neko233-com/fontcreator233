// Package ui defines build progress events and terminal reporters.
package ui

import "time"

// EventKind identifies a progress event.
type EventKind int

const (
	EvPhase EventKind = iota
	EvProgress
	EvCurrent
	EvLog
	EvDone
	EvError
)

// Event is a single progress update.
type Event struct {
	Kind     EventKind
	Phase    string
	Detail   string
	Current  int
	Total    int
	Err      error
	Warnings []string
	Duration time.Duration
}

// Reporter consumes build events. Implementations must be safe for concurrent
// calls.
type Reporter interface {
	Event(Event)
}

// NopReporter drops all events.
type NopReporter struct{}

// Event implements Reporter.
func (NopReporter) Event(Event) {}
