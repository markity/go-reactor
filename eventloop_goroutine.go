package goreactor

import (
	"sync/atomic"

	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

type eventloopGoroutine struct {
	started int64
}

func (routine *eventloopGoroutine) startLoop() eventloop.EventLoop {
	if !atomic.CompareAndSwapInt64(&routine.started, 0, 1) {
		panic("already started")
	}

	c := make(chan eventloop.EventLoop, 1)
	go func() {
		loop := eventloop.NewEventLoop()
		c <- loop
		loop.Loop()
	}()

	return <-c
}

func newEventLoopGoroutine() *eventloopGoroutine {
	return &eventloopGoroutine{
		started: 0,
	}
}
