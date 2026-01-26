package watcher

import (
	"sync"
	"time"
)

type Debouncer struct {
	window   time.Duration
	maxBatch int
	events   map[string]FileEvent
	mu       sync.Mutex
	timer    *time.Timer
	onFlush  func([]FileEvent)
	stopped  bool
}

func NewDebouncer(window time.Duration, maxBatch int, onFlush func([]FileEvent)) *Debouncer {
	return &Debouncer{
		window:   window,
		maxBatch: maxBatch,
		events:   make(map[string]FileEvent),
		onFlush:  onFlush,
	}
}

func (d *Debouncer) Add(event FileEvent) {
	d.mu.Lock()

	if d.stopped {
		d.mu.Unlock()
		return
	}

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	d.events[event.Path] = event

	if len(d.events) >= d.maxBatch {
		d.flushLocked()
		return
	}

	d.timer = time.AfterFunc(d.window, func() {
		d.mu.Lock()
		if !d.stopped {
			d.flushLocked()
		} else {
			d.mu.Unlock()
		}
	})

	d.mu.Unlock()
}

func (d *Debouncer) flushLocked() {
	events := make([]FileEvent, 0, len(d.events))
	for _, event := range d.events {
		events = append(events, event)
	}

	d.events = make(map[string]FileEvent)

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	d.mu.Unlock()

	if len(events) > 0 && d.onFlush != nil {
		d.onFlush(events)
	}
}

func (d *Debouncer) Stop() {
	d.mu.Lock()

	if d.stopped {
		d.mu.Unlock()
		return
	}

	d.stopped = true

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	if len(d.events) > 0 {
		d.flushLocked()
	} else {
		d.mu.Unlock()
	}
}
