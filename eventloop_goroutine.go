package goreactor

import (
	"sync/atomic"

	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

type eventloopGoroutine struct {
	started int64
	loop    eventloop.EventLoop
}

func (routine *eventloopGoroutine) startLoop() {
	if !atomic.CompareAndSwapInt64(&routine.started, 0, 1) {
		panic("already started")
	}

	go func() {
		routine.loop.Loop()
	}()
}

func newEventLoopGoroutine() *eventloopGoroutine {
	loop := eventloop.NewEventLoop()
	return &eventloopGoroutine{
		started: 0,
		loop:    loop,
	}
}
