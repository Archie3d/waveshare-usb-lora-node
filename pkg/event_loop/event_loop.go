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

	mutex          sync.Mutex
	eventQueue     *eventPoint
	eventQueueTail *eventPoint
}

func NewEventLoop() EventLoop {
	ctx, cancel := context.WithCancel(context.Background())
	return &event_loop{
		ctx:        ctx,
		cancel:     cancel,
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
			// event has expired - execute it
			event.callback(el)
		} else {
			// event cannot be scheduled just yet
			postponeBy := time.Until(event.scheduledBy)
			if sleepDuration < 0 || postponeBy < sleepDuration {
				sleepDuration = postponeBy
			}

			el.Post(event.callback, event.scheduledBy)
		}

		// Move to the next event
		event = event.next
	}

	return sleepDuration
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
