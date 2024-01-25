package eventloop

import (
	"errors"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type eventloop struct {
	poller Poller

	wakeupEventFD      int
	wakeupEventChannel Channel

	timerQueue *timerQueue

	mu       sync.Mutex
	functors []func()
	running  int64

	// if running gid is routine id
	gid uint64
}

func NewEventLoop() EventLoop {
	r1, _, errn := syscall.Syscall(syscall.SYS_EVENTFD, 0, 0, 0)
	if errn != 0 {
		panic(errn)
	}

	err := syscall.SetNonblock(int(r1), true)
	if err != nil {
		panic(err)
	}

	c := NewChannel(int(r1))
	c.SetEvent(ReadableEvent)
	c.SetReadCallback(func() {
		_, err := syscall.Read(int(r1), make([]byte, 8))
		if err != nil {
			panic(err)
		}
	})

	ev := &eventloop{
		poller:             NewPoller(),
		wakeupEventFD:      int(r1),
		wakeupEventChannel: c,
		functors:           make([]func(), 0),
		running:            0,
		gid:                getGid(),
	}
	ev.UpdateChannelInLoopGoroutine(ev.wakeupEventChannel)

	ev.timerQueue = newTimerQueue(ev)

	return ev
}

type EventLoop interface {
	Loop()
	RunInLoop(func())
	Stop()
	RunAt(triggerAt time.Time, interval time.Duration, f func())

	UpdateChannelInLoopGoroutine(Channel)
	RemoveChannelInLoopGoroutine(Channel)
	RunAtInLoopGoroutine(triggerAt time.Time, interval time.Duration, f func()) int
	CancelTimerInLoopGoroutine(id int) bool
}

// 非异步安全
func (ev *eventloop) UpdateChannelInLoopGoroutine(c Channel) {
	ev.poller.UpdateChannel(c)
}

// 非异步安全
func (ev *eventloop) RemoveChannelInLoopGoroutine(c Channel) {
	ev.poller.RemoveChannel(c)
}

// 非异步安全
func (ev *eventloop) Loop() {
	if ev.gid != getGid() {
		panic("loop must be run at the goroutine created at")
	}
	ev.mu.Lock()
	atomic.StoreInt64(&ev.running, 1)
	ev.mu.Unlock()

	for atomic.LoadInt64(&ev.running) == 1 {
		channels := ev.poller.Poll()
		for _, v := range channels {
			v.HandleEvent()
		}
		ev.mu.Lock()
		f := ev.functors
		ev.functors = make([]func(), 0)
		ev.mu.Unlock()
		for _, v := range f {
			v()
		}
	}
}

// 异步安全
func (ev *eventloop) RunInLoop(f func()) {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.running == 1 && ev.gid == getGid() {
		f()
	} else {
		ev.functors = append(ev.functors, f)
		_, err := syscall.Write(ev.wakeupEventChannel.GetFD(), []byte{0, 0, 0, 0, 0, 0, 0, 1})
		if err != nil {
			if !errors.Is(err, syscall.EAGAIN) {
				panic(err)
			}
		}
	}
}

// 异步安全
func (ev *eventloop) Stop() {
	ev.RunInLoop(func() {
		ev.running = 0
	})
}

// 异步安全
func (ev *eventloop) RunAt(triggerAt time.Time, interval time.Duration, f func()) {
	ev.RunInLoop(func() {
		ev.timerQueue.AddTimer(triggerAt, interval, f)
	})
}

// 非异步安全, 只能在io协程使用
func (ev *eventloop) RunAtInLoopGoroutine(triggerAt time.Time, interval time.Duration, f func()) int {
	return ev.timerQueue.AddTimer(triggerAt, interval, f)
}

// 非异步安全, 只能在io协程使用
func (ev *eventloop) CancelTimerInLoopGoroutine(id int) bool {
	return ev.timerQueue.CancelTimer(id)
}
