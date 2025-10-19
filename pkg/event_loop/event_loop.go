package event_loop

import (
	"context"
	"sync"
	"time"
)

type CallbackFunc func(el EventLoop)

type EventLoop interface {
	Run()
	Quit()
	Put(callback CallbackFunc)
	Post(callback CallbackFunc, scheduledBy time.Time)
}

type eventPoint struct {
	callback    CallbackFunc
	scheduledBy time.Time
	next        *eventPoint
}

type event_loop struct {
	ctx    context.Context
	cancel context.CancelFunc

	wake chan bool

	mutex          sync.Mutex
	eventQueue     *eventPoint
	eventQueueTail *eventPoint
}

func NewEventLoop() EventLoop {
	ctx, cancel := context.WithCancel(context.Background())
	return &event_loop{
		ctx:        ctx,
		cancel:     cancel,
		wake:       make(chan bool, 1),
		eventQueue: nil,
	}
}

func (el *event_loop) Run() {
	var sleepDuration time.Duration = 0

loop:
	for {
		select {
		case <-el.ctx.Done():
			break loop
		case <-el.wake:
			sleepDuration = el.processEvents()
		case <-time.After(sleepDuration):
			sleepDuration = el.processEvents()
		}
	}
}

func (el *event_loop) processEvents() time.Duration {
	el.mutex.Lock()
	event := el.eventQueue
	el.eventQueue = nil
	el.eventQueueTail = nil
	el.mutex.Unlock()

	var sleepDuration time.Duration = -1

	for event != nil {

		if time.Since(event.scheduledBy) > 0 {
			// Event has expired - execute it
			event.callback(el)

			// Event callback may produce more events, so we have
			// do cancel sleep here
			sleepDuration = time.Duration(0)
		} else {
			// event cannot be scheduled just yet
			postponeBy := time.Until(event.scheduledBy)
			if sleepDuration < 0 || postponeBy < sleepDuration {
				sleepDuration = postponeBy
			}

			el.enqueue(event)
		}

		// Move to the next event
		event = event.next
	}

	if sleepDuration < 0 {
		// No events, sleep
		sleepDuration = 100 * time.Millisecond
	}

	return sleepDuration
}

func (el *event_loop) enqueue(event *eventPoint) {
	el.mutex.Lock()
	defer el.mutex.Unlock()

	if el.eventQueue == nil {
		el.eventQueue = event
		el.eventQueueTail = event
	} else {
		el.eventQueueTail.next = event
		el.eventQueueTail = el.eventQueueTail.next
	}
}

func (el *event_loop) wakeUp() {
	select {
	case el.wake <- true:
	default:
		// Already woken up
	}
}

func (el *event_loop) Quit() {
	if el.cancel != nil {
		el.cancel()
	}
}

func (el *event_loop) Put(callback CallbackFunc) {
	el.Post(callback, time.Now())
}

func (el *event_loop) Post(callback CallbackFunc, scheduledBy time.Time) {
	event := &eventPoint{
		callback:    callback,
		scheduledBy: scheduledBy,
		next:        nil,
	}

	el.enqueue(event)
	el.wakeUp()
}
