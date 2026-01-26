package watcher

import (
	"time"
)

type EventType int

const (
	EventCreate EventType = iota
	EventModify
	EventDelete
	EventRename
)

func (e EventType) String() string {
	switch e {
	case EventCreate:
		return "create"
	case EventModify:
		return "modify"
	case EventDelete:
		return "delete"
	case EventRename:
		return "rename"
	default:
		return "unknown"
	}
}

type FileEvent struct {
	Path      string
	Type      EventType
	Timestamp time.Time
}

type EventClassifier struct{}

func NewEventClassifier() *EventClassifier {
	return &EventClassifier{}
}

func (c *EventClassifier) ClassifyBatch(events []FileEvent) int {
	count := len(events)

	if count > 10 {
		return 0
	}

	if count >= 3 {
		return 1
	}

	return 2
}
