package eventloop

import (
	"errors"
	"syscall"
)

type poller struct {
	// epoll file descriptor
	epollFD int

	// key is fd, value is Channel
	channelMap map[int]Channel

	// params of epoll_wait, it can expand if one epoll_wait syscall fills up this array
	eventList []syscall.EpollEvent

	// for each channel, assign a index, the first index is 1
	idxCounter int
}

// create a new poller, poller contains a epollfd,
func NewPoller() Poller {
	// size is ignored since linux 2.6.8, see man epoll_create(2)
	fd, err := syscall.EpollCreate(1)
	if err != nil {
		panic(err)
	}

	return &poller{
		epollFD:    fd,
		channelMap: make(map[int]Channel),
		eventList:  make([]syscall.EpollEvent, 128),
		idxCounter: 0,
	}
}

type Poller interface {
	// wait on epoll_wait and returns the active Channels
	Poll() []Channel

	// UpdateChannel calls epoll_ctl
	UpdateChannel(Channel)

	// RemoveChannel removes a fd from epollfd
	RemoveChannel(Channel)

	// GetChannelCount get current epoll wait fd nums, may be used to implement load balance
	GetChannelCount() int
}

// wait on epoll_wait and returns the active Channels
func (p *poller) Poll() []Channel {
	var n int

	for {
		var err error
		n, err = syscall.EpollWait(p.epollFD, p.eventList, -1)
		if err != nil {
			// golang send signals to implement signal preemption even if we are blocking
			// on syscall, it is necessary to distingulish the error
			if errors.Is(err, syscall.EINTR) {
				continue
			}
			panic(err)
		}
		break
	}

	// fill up active channels
	c := make([]Channel, 0)
	if n > 0 {
		for i := 0; i < n; i++ {
			ch := p.channelMap[int(p.eventList[i].Fd)]
			ch.SetRevent(ReactorEvent(p.eventList[i].Events))
			c = append(c, p.channelMap[int(p.eventList[i].Fd)])
		}
	}

	// if eventList is full, expand the array
	if n == len(p.eventList) {
		p.eventList = make([]syscall.EpollEvent, 2*len(p.eventList))
	}

	return c
}

// UpdateChannel calls epoll_ctl
func (p *poller) UpdateChannel(c Channel) {
	// if index == -1, it is a new channel, assign a new index for it
	if c.GetIndex() < 0 {
		// register fd into epollfd
		err := syscall.EpollCtl(p.epollFD, syscall.EPOLL_CTL_ADD, c.GetFD(), &syscall.EpollEvent{
			Events: uint32(c.GetEvent()),
			Fd:     int32(c.GetFD()),
		})
		if err != nil {
			panic(err)
		}

		// assign an index
		p.idxCounter++
		p.channelMap[c.GetFD()] = c
		c.SetIndex(p.idxCounter)
	} else {
		// channel exists, now just modify epollfd
		err := syscall.EpollCtl(p.epollFD, syscall.EPOLL_CTL_MOD, c.GetFD(), &syscall.EpollEvent{
			Events: uint32(c.GetEvent()),
			Fd:     int32(c.GetFD()),
		})
		if err != nil {
			panic(err)
		}
	}
}

// RemoveChannel removes a fd from epollfd
func (p *poller) RemoveChannel(c Channel) {
	if c.GetIndex() < 0 {
		panic("remove non-exist channel")
	}

	err := syscall.EpollCtl(p.epollFD, syscall.EPOLL_CTL_DEL, c.GetFD(), nil)
	if err != nil {
		panic(err)
	}

	delete(p.channelMap, c.GetFD())
}

func (p *poller) GetChannelCount() int {
	return len(p.channelMap)
}
