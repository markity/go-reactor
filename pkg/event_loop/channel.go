package eventloop

import "syscall"

type ReactorEvent int

const (
	NoneEvent     ReactorEvent = 0
	ReadableEvent ReactorEvent = syscall.EPOLLIN
	WritableEvent ReactorEvent = syscall.EPOLLOUT
	ErrorEvent    ReactorEvent = syscall.EPOLLERR
	CloseEvent    ReactorEvent = syscall.EPOLLHUP
)

type channel struct {
	fd     int
	events ReactorEvent

	// revents是此eventfd的epoll结果
	revents ReactorEvent
	index   int // used by poller

	readCallback  func()
	writeCallback func()
	errorCallback func()
	closeCallback func()
}

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

// bool代表是否修修改成功
func (c *channel) EnableWrite() bool {
	if c.events&WritableEvent != 0 {
		return false
	}

	c.events |= WritableEvent
	return true
}

// bool代表是否修改成功
func (c *channel) DisableWrite() bool {
	if c.events&WritableEvent == 0 {
		return false
	}

	c.events &= ^WritableEvent
	return true
}

// bool代表是否修修改成功
func (c *channel) EnableRead() bool {
	if c.events&ReadableEvent != 0 {
		return false
	}

	c.events |= ReadableEvent
	return true
}

// bool代表是否修改成功
func (c *channel) DisableRead() bool {
	if c.events&ReadableEvent == 0 {
		return false
	}

	c.events &= ^ReadableEvent
	return true
}

func (c *channel) HandleEvent() {
	// 这种情况说来复杂, 我没有实践过, 为了保险需要判断
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

func NewChannel(fd int) Channel {
	return &channel{
		index: -1,
		fd:    fd,
	}
}

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
