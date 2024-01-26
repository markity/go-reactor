package goreactor

import (
	eventloop "go-reactor/pkg/event_loop"
)

type eventloopGoroutinePoll struct {
	numOfGoroutine int
	started        int64
	loops          []eventloop.EventLoop
	baseLoop       eventloop.EventLoop
	strategy       LoadBalanceStrategy
}

func newEventloopGoroutinePoll(baseLoop eventloop.EventLoop,
	numOfGoroutinePoll int, strategy LoadBalanceStrategy) *eventloopGoroutinePoll {
	if numOfGoroutinePoll < 0 {
		panic(numOfGoroutinePoll)
	}

	return &eventloopGoroutinePoll{
		numOfGoroutine: numOfGoroutinePoll,
		started:        0,
		baseLoop:       baseLoop,
		strategy:       strategy,
	}
}

func (poll *eventloopGoroutinePoll) start() {
	if poll.started == 1 {
		panic(poll.started)
	}

	loops := make([]eventloop.EventLoop, 0, poll.numOfGoroutine)
	for i := 1; i <= poll.numOfGoroutine; i++ {
		g := newEventLoopGoroutine(i)
		loops = append(loops, g.startLoop())
	}
	poll.loops = loops
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
