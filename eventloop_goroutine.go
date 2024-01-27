package goreactor

import (
	"sync/atomic"

	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

type eventloopGoroutine struct {
	started int64
	id      int
}

func (routine *eventloopGoroutine) startLoop() eventloop.EventLoop {
	if !atomic.CompareAndSwapInt64(&routine.started, 0, 1) {
		panic("already started")
	}

	c := make(chan eventloop.EventLoop, 1)
	go func() {
		loop := eventloop.NewEventLoop(routine.id)
		c <- loop
		loop.Loop()
	}()

	return <-c
}

func newEventLoopGoroutine(id int) *eventloopGoroutine {
	return &eventloopGoroutine{
		started: 0,
		id:      id,
	}
}
