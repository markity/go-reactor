package eventloop

import (
	"errors"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	kvcontext "github.com/markity/go-reactor/pkg/context"
)

type eventloop struct {
	// Poller is epoll poller
	poller Poller

	// eventfd, be used to wake up epoll_wait syscall
	wakeupEventChannel Channel

	// be used to manage timers, timerQueue contains a timerfd
	timerQueue *timerQueue

	// mu is used to protect functors
	mu       sync.Mutex
	functors []func()

	// be used to stop eventloop, make eventloop.Loop returns
	running int64

	// gid is goroutine id, be set when NewEventLoop
	gid uint64

	id int

	ctx kvcontext.KVContext

	doOnLoop func(EventLoop)
}

// create an EventLoop, it's Loop function can be only triggered
// at the goroutine which creates the eventloop
func NewEventLoop() EventLoop {
	// create an eventfd, the eventfd is used to wake up epoll_wait

	// for example, when a loop.RunInLoop(f) is called, but now the
	// loop goroutine is waiting on the epoll_wait() syscall, we need to
	// notify that there is a functor to call. after putting a functor into
	// loop.functors, we need to write to eventfd to notify evloop to handle
	// the functors
	r1, _, errno := syscall.Syscall(syscall.SYS_EVENTFD, 0, 0, 0)
	if errno != 0 {
		panic(errno)
	}

	// to make sure syscall.Read(eventfd) will not block, it is not necessary actually,
	// but sometimes the user may misread our eventfd by mistake, after witch reading
	// eventfd will make the loop goroutine block
	err := syscall.SetNonblock(int(r1), true)
	if err != nil {
		panic(err)
	}

	// create a channel for eventfd
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
		wakeupEventChannel: c,
		functors:           make([]func(), 0),
		running:            0,
		gid:                getGid(),
		id:                 int(idGen.Add(1)),
		ctx:                kvcontext.NewContext(),
	}

	// register the channel into epoll
	ev.UpdateChannelInLoopGoroutine(ev.wakeupEventChannel)

	// create a timer queue
	ev.timerQueue = newTimerQueue(ev)

	return ev
}

// EventLoop interface describe the functions designed for the users
type EventLoop interface {
	// Loop() will never return unless Stop() is called
	Loop()

	// queue a functor into eventloop, the function will be called latter in loop goroutine
	RunInLoop(func())

	// stop eventloop and make Loop() return
	Stop()

	// create a timer, it will be triggered at specified timepoint
	RunAt(triggerAt time.Time, interval time.Duration, f func()) int

	// cancel a timer, if it is removed successfully, returns true
	// if the timer is already executed or the id is invalid, returns false
	CancelTimer(id int) bool

	// for tcp server, each eventloop has its id, it may be used by users
	GetID() int

	// get current channel count in this loop, it may be used for load balance
	GetChannelCount() int

	// for go-reactor developers, this is be used to register channel into epollfd
	// go-reacotr users can ignore functions below

	// when a channel is change, it is necessary to notify epollfd
	UpdateChannelInLoopGoroutine(Channel)

	// remove a channel from eventloop, the fd will also be remove from epollfd
	RemoveChannelInLoopGoroutine(Channel)

	// about kv context
	GetContext(key string) (interface{}, bool)
	SetContext(key string, val interface{})
	DeleteContext(key string)

	// be called when start loop
	DoOnLoop(func(EventLoop))
}

func (ev *eventloop) GetContext(key string) (interface{}, bool) {
	return ev.ctx.Get(key)
}

func (ev *eventloop) SetContext(key string, val interface{}) {
	ev.ctx.Set(key, val)
}

func (ev *eventloop) DeleteContext(key string) {
	ev.ctx.Delete(key)
}

// the function can be only triggered at eventloop goroutine
func (ev *eventloop) UpdateChannelInLoopGoroutine(c Channel) {
	ev.poller.UpdateChannel(c)
}

// the function can be only triggered at eventloop goroutine
func (ev *eventloop) RemoveChannelInLoopGoroutine(c Channel) {
	ev.poller.RemoveChannel(c)
}

func (ev *eventloop) DoOnLoop(f func(EventLoop)) {
	ev.doOnLoop = f
}

// start event loop, if Stop() is not called, Loop() will never return
func (ev *eventloop) Loop() {
	// check gid, Loop() can be only called at the goroutine which creates it
	if ev.gid != getGid() {
		panic("loop must be run at the goroutine created at")
	}

	// atomic operation, make running switch 0 to 1
	if !atomic.CompareAndSwapInt64(&ev.running, 0, 1) {
		panic("it is already running? don't run it again")
	}

	if ev.doOnLoop != nil {
		ev.doOnLoop(ev)
	}

	// check running, if running is 0, Loop should returns
	for atomic.LoadInt64(&ev.running) == 1 {
		// wait epoll_wait returns, and get the active event channels
		channels := ev.poller.Poll()

		// execute functions for each channel
		for _, v := range channels {
			v.HandleEvent()
		}

		// get all functors
		ev.mu.Lock()
		f := ev.functors
		ev.functors = make([]func(), 0)
		ev.mu.Unlock()

		// execute all functors
		for _, v := range f {
			v()
		}
	}
}

// queue a functor into a loop, func will be called in the loop goroutine later
func (ev *eventloop) RunInLoop(f func()) {
	// if is running and it is in eventloop goroutine, just execute it right now
	if running := atomic.LoadInt64(&ev.running) == 1; running && ev.gid == getGid() {
		f()
	} else {
		// or queue the functor into ev.functors, the lock protects functors
		ev.mu.Lock()
		ev.functors = append(ev.functors, f)
		ev.mu.Unlock()

		// make sure epoll_wait returns
		ev.wakeup()
	}
}

// stop a eventloop
func (ev *eventloop) Stop() {
	ev.RunInLoop(func() {
		// atomic operation is better than lock
		atomic.StoreInt64(&ev.running, 0)
	})
}

func (ev *eventloop) GetID() int {
	return ev.id
}

func (ev *eventloop) GetChannelCount() int {
	if getGid() == ev.gid {
		return ev.poller.GetChannelCount()
	}

	c := make(chan int, 1)
	ev.RunInLoop(func() {
		c <- ev.poller.GetChannelCount()
	})
	return <-c
}

// setup a timer, returns its id, it can be cancelled, see CancelTimer(id int)
func (ev *eventloop) RunAt(triggerAt time.Time, interval time.Duration, f func()) int {
	// ev.timerQueue can noly be operated in loop goroutine, we need to use RunInLoop
	// and get its return value by golang channel
	if getGid() == ev.gid {
		return ev.timerQueue.AddTimer(triggerAt, interval, f)
	}

	c := make(chan int, 1)
	ev.RunInLoop(func() {
		c <- ev.timerQueue.AddTimer(triggerAt, interval, f)
	})
	return <-c
}

// cancel a timer
func (ev *eventloop) CancelTimer(id int) bool {
	if getGid() == ev.gid {
		return ev.timerQueue.CancelTimer(id)
	}

	c := make(chan bool, 1)
	ev.RunInLoop(func() {
		c <- ev.timerQueue.CancelTimer(id)
	})
	return <-c
}

// wakeup writes something into evnetfd, so that epoll_wait can return
func (ev *eventloop) wakeup() {
	_, err := syscall.Write(ev.wakeupEventChannel.GetFD(), []byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		// notice here, if the content of eventfd is not comsumed, write will returns EAGAIN
		// for example, double writes to an eventfd will trigger EAGAIN
		if !errors.Is(err, syscall.EAGAIN) {
			panic(err)
		}
	}
}
