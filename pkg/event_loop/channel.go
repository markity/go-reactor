package eventloop

type ReactorEvent int

const (
	// do not care about anything
	NoneEvent     ReactorEvent = 0
	ReadableEvent ReactorEvent = 0b1
	WritableEvent ReactorEvent = 0b10
	AllEvent      ReactorEvent = ReadableEvent | WritableEvent
)

type channel struct {
	//  file descripor, each channel is used only to handle one fd
	fd int

	// events that we are interested, if we want to do something when the fd
	// is readable, we need to set events to ReadableEvent and SetReadCallback
	events    ReactorEvent
	tobeWrite []byte
	// is isAccept is true, use uring.Accept instead of uring.Read
	isAccept bool

	revents                ReactorEvent
	handleEventReadCbData1 []byte
	handleEventReadCbData2 int

	writePending bool // write op submitted
	readPending  bool // read op submitted

	handleEventWriteCbData int

	// used by poller, if index is zero, the poller knows it is a new channel,
	// and poller will set a index for the channel
	index int

	// callbacks
	readCallback  func([]byte, int)
	writeCallback func(int)
}

// some setters and getters

func (c *channel) GetEvent() ReactorEvent {
	return c.events
}

func (c *channel) SetEvent(e ReactorEvent) {
	c.events = e
}

func (c *channel) IsWritePending() bool {
	return c.writePending
}

func (c *channel) IsReadPending() bool {
	return c.readPending
}

func (c *channel) EnableReventRead(bs []byte, res int) {
	c.revents |= ReadableEvent
	c.handleEventReadCbData1 = bs
	c.handleEventReadCbData2 = res
}

func (c *channel) EnableReventWrite(d int) {
	c.revents |= WritableEvent
	c.handleEventWriteCbData = d
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

func (c *channel) SetReadCallback(f func([]byte, int)) {
	c.readCallback = f
}

func (c *channel) SetWriteCallback(f func(int)) {
	c.writeCallback = f
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
func (c *channel) EnableWrite(bs []byte) bool {
	if c.events&WritableEvent != 0 {
		return false
	}

	c.events |= WritableEvent
	c.tobeWrite = bs
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
func (c *channel) EnableRead(isAccept bool) bool {
	if c.events&ReadableEvent != 0 {
		return false
	}

	c.events |= ReadableEvent
	c.isAccept = isAccept
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
	// TODO: EPOLLHUP

	if c.revents&ReadableEvent != 0 {
		if c.readCallback != nil {
			c.readCallback(c.handleEventReadCbData1, c.handleEventReadCbData2)
		}
	}

	if c.revents&WritableEvent != 0 {
		if c.writeCallback != nil {
			c.writeCallback(c.handleEventWriteCbData)
		}
	}

	c.revents = 0
}

func (c *channel) EnableWritePending() {
	c.writePending = true
}

func (c *channel) DisableWritePending() {
	c.writePending = false
}

func (c *channel) EnableReadPending() {
	c.readPending = true
}

func (c *channel) DisableReadPending() {
	c.readPending = false
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

	GetIndex() int
	SetIndex(int)

	GetFD() int

	SetReadCallback(func([]byte, int))
	SetWriteCallback(func(int))

	HandleEvent()

	IsWriting() bool
	IsReading() bool
	EnableWrite([]byte) bool
	DisableWrite() bool
	EnableRead(isAccept bool) bool
	DisableRead() bool

	IsWritePending() bool
	IsReadPending() bool

	EnableWritePending()
	DisableWritePending()
	EnableReadPending()
	DisableReadPending()
}
