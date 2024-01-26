package eventloop

import (
	"syscall"
)

type ReactorEvent int

const (
	// do not care about anything
	NoneEvent     ReactorEvent = 0
	ReadableEvent ReactorEvent = syscall.EPOLLIN
	WritableEvent ReactorEvent = syscall.EPOLLOUT
	ErrorEvent    ReactorEvent = syscall.EPOLLERR
	CloseEvent    ReactorEvent = syscall.EPOLLHUP
	AllEvent      ReactorEvent = ReadableEvent | WritableEvent | ErrorEvent | CloseEvent
)

type channel struct {
	// file descripor, each channel is used only for handle one fd
	fd int

	// events that we are interested, if we want to do something when the fd
	// is readable, we need to set events to ReadableEvent and SetReadCallback
	events ReactorEvent

	// revents tells what events are active, for example, after the fd is readable
	// poller.Poll() returns the channel with (revents & ReadableEvent == 1)
	revents ReactorEvent

	// used by poller, if index is zero, the poller knows it is a new channel,
	// and poller will set a index for the channel
	index int

	// callbacks
	readCallback  func()
	writeCallback func()
	errorCallback func()
	closeCallback func()
}

// some setters and getters

func (c *channel) GetEvent() ReactorEvent {
	return c.events
}

func (c *channel) SetEvent(e ReactorEvent) {
	c.events = e
}

func (c *channel) GetRevent() ReactorEvent {
	return c.revents
}

func (c *channel) SetRevent(e ReactorEvent) {
	c.revents = e
}

func (c *channel) GetIndex() int {
	return c.index
}

func (c *channel) SetIndex(i int) {
	c.index = i
}

func (c *channel) GetFD() int {
	return c.fd
}

func (c *channel) SetReadCallback(f func()) {
	c.readCallback = f
}

func (c *channel) SetWriteCallback(f func()) {
	c.writeCallback = f
}

func (c *channel) SetErrorCallback(f func()) {
	c.errorCallback = f
}

func (c *channel) SetCloseCallback(f func()) {
	c.closeCallback = f
}

func (c *channel) IsWriting() bool {
	return c.events&WritableEvent != 0
}

func (c *channel) IsReading() bool {
	return c.events&ReadableEvent != 0
}

// make events with WritableEvent set, if WritableEvent is already set before the call
// returns false, it is used for better performance, when we call EnableWrite() with
// false returns, we do not need to call eventloop.UpdateChannelInLoopGoroutine, this
// save the cost of the epoll_ctl system call
func (c *channel) EnableWrite() bool {
	if c.events&WritableEvent != 0 {
		return false
	}

	c.events |= WritableEvent
	return true
}

// if WritableEvent is not set before, returns false, the return value is be used to
// save the cost of the epoll_ctl, see EnableWrite comments
func (c *channel) DisableWrite() bool {
	if c.events&WritableEvent == 0 {
		return false
	}

	c.events &= ^WritableEvent
	return true
}

// enable read
func (c *channel) EnableRead() bool {
	if c.events&ReadableEvent != 0 {
		return false
	}

	c.events |= ReadableEvent
	return true
}

// disable read
func (c *channel) DisableRead() bool {
	if c.events&ReadableEvent == 0 {
		return false
	}

	c.events &= ^ReadableEvent
	return true
}

// handle all events for the channel
func (c *channel) HandleEvent() {
	// TODO: more comment
	if (c.revents&syscall.EPOLLHUP) != 0 && (c.revents&syscall.EPOLLIN) == 0 {
		if c.closeCallback != nil {
			c.closeCallback()
		}
	}

	if c.revents&syscall.EPOLLERR != 0 {
		if c.errorCallback != nil {
			c.errorCallback()
		}
	}

	if c.revents&ReadableEvent != 0 {
		if c.readCallback != nil {
			c.readCallback()
		}
	}

	if c.revents&WritableEvent != 0 {
		if c.writeCallback != nil {
			c.writeCallback()
		}
	}
}

// create a new channel
func NewChannel(fd int) Channel {
	return &channel{
		index: -1,
		fd:    fd,
	}
}

// Channel is used to manage a fd events, and handle callbacks
type Channel interface {
	GetEvent() ReactorEvent
	SetEvent(ReactorEvent)

	GetRevent() ReactorEvent
	SetRevent(ReactorEvent)

	GetIndex() int
	SetIndex(int)

	GetFD() int

	SetReadCallback(func())
	SetWriteCallback(func())
	SetErrorCallback(func())
	SetCloseCallback(func())

	HandleEvent()

	IsWriting() bool
	IsReading() bool
	EnableWrite() bool
	DisableWrite() bool
	EnableRead() bool
	DisableRead() bool
}
