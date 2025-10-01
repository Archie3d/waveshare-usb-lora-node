package event_loop

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	eventLoop := NewEventLoop()

	done := false

	trigger := make(chan bool)

	go func() {
		<-trigger
		eventLoop.Run()
		done = true
	}()

	assert.False(t, done)
	trigger <- true

	// Wait for goroutine to start
	<-time.After(100 * time.Millisecond)

	eventLoop.Quit()

	// Wait for goroutine to finish
	<-time.After(100 * time.Millisecond)

	assert.True(t, done)
}

func TestPutEvents(t *testing.T) {
	eventLoop := NewEventLoop()

	go eventLoop.Run()

	var counter atomic.Int32

	eventLoop.Put(func(el EventLoop) {
		counter.Add(1)

		el.Put(func(el EventLoop) {
			counter.Add(2)

			el.Quit()
		})
	})

	<-time.After(100 * time.Millisecond)

	assert.Equal(t, int32(3), counter.Load())
}

func TestPostEvents(t *testing.T) {
	eventLoop := NewEventLoop()

	done := make(chan bool, 1)
	go func() {
		eventLoop.Run()
		done <- true
	}()
	defer eventLoop.Quit()

	now := time.Now()

	var mutex sync.Mutex
	var timepoints []time.Time
	var expectedTimepoints []time.Time

	for i := range 5 {
		scheduledBy := now.Add(time.Duration(i) * time.Second)
		eventLoop.Post(func(el EventLoop) {
			mutex.Lock()
			defer mutex.Unlock()
			timepoints = append(timepoints, time.Now())

			if i == 4 {
				// Last event to terminate the event loop
				el.Quit()
			}
		}, scheduledBy)

		expectedTimepoints = append(expectedTimepoints, scheduledBy)
	}

	// Wait for the events to process
	select {
	case <-done:
		// All good
	case <-time.After(5 * time.Second):
		t.Errorf("Event loop is still running")
	}

	// We are expecting 5 timepoints to be recorded
	assert.Equal(t, 5, len(timepoints))

	for i := range 5 {
		assert.True(t, expectedTimepoints[i].Sub(timepoints[i]).Abs() < 10*time.Millisecond)
	}
}
