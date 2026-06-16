package mobile

import "time"

const defaultEventQueueSize = 128

type Event struct {
	AtUnixMillis int64
	Level        string
	Source       string
	Message      string
}

type EventQueue struct {
	ch chan *Event
}

func NewEventQueue(capacity int) *EventQueue {
	if capacity <= 0 {
		capacity = defaultEventQueueSize
	}
	return &EventQueue{ch: make(chan *Event, capacity)}
}

func (q *EventQueue) Next(timeoutMs int) *Event {
	if q == nil {
		return nil
	}
	if timeoutMs < 0 {
		event := <-q.ch
		return event
	}
	if timeoutMs == 0 {
		select {
		case event := <-q.ch:
			return event
		default:
			return nil
		}
	}

	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timer.Stop()
	select {
	case event := <-q.ch:
		return event
	case <-timer.C:
		return nil
	}
}

func (q *EventQueue) Len() int {
	if q == nil {
		return 0
	}
	return len(q.ch)
}

func (q *EventQueue) emit(level, source, message string) {
	if q == nil {
		return
	}
	event := &Event{
		AtUnixMillis: time.Now().UnixMilli(),
		Level:        level,
		Source:       source,
		Message:      message,
	}
	select {
	case q.ch <- event:
		return
	default:
	}
	select {
	case <-q.ch:
	default:
	}
	select {
	case q.ch <- event:
	default:
	}
}
