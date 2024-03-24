package goreactor

import (
	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

type eventloopGoroutinePoll struct {
	numOfGoroutine int
	started        int64
	loopGoroutines []*eventloopGoroutine
	loops          []eventloop.EventLoop
	baseLoop       eventloop.EventLoop
	strategy       LoadBalanceStrategy
}

func newEventloopGoroutinePoll(baseLoop eventloop.EventLoop,
	numOfGoroutinePoll int, strategy LoadBalanceStrategy) *eventloopGoroutinePoll {
	if numOfGoroutinePoll < 0 {
		panic(numOfGoroutinePoll)
	}

	loopGoroutines := make([]*eventloopGoroutine, 0, numOfGoroutinePoll)
	loops := make([]eventloop.EventLoop, 0, numOfGoroutinePoll)
	for i := 1; i <= numOfGoroutinePoll; i++ {
		g := newEventLoopGoroutine()
		loopGoroutines = append(loopGoroutines, g)
		loops = append(loops, g.loop)
	}

	return &eventloopGoroutinePoll{
		numOfGoroutine: numOfGoroutinePoll,
		started:        0,
		baseLoop:       baseLoop,
		strategy:       strategy,
		loopGoroutines: loopGoroutines,
		loops:          loops,
	}
}

func (poll *eventloopGoroutinePoll) start() {
	if poll.started == 1 {
		panic(poll.started)
	}

	for _, loop := range poll.loopGoroutines {
		loop.startLoop()
	}

	poll.started = 1
}

func (poll *eventloopGoroutinePoll) getNext() eventloop.EventLoop {
	if poll.started == 0 {
		panic("not started yet")
	}

	loop := poll.baseLoop
	if poll.numOfGoroutine != 0 {
		loop = poll.strategy(poll.loops)
	}

	return loop
}
