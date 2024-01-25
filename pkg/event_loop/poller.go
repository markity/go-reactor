package eventloop

import (
	"errors"
	"syscall"
)

type poller struct {
	epollFD    int
	channelMap map[int]Channel
	eventList  []syscall.EpollEvent

	idxCounter int
}

func NewPoller() Poller {
	// size is ignored
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
	Poll() []Channel
	UpdateChannel(Channel)
	RemoveChannel(Channel)
}

func (p *poller) Poll() []Channel {
	var n int
	for {
		var err error
		n, err = syscall.EpollWait(p.epollFD, p.eventList, -1)
		if err != nil {
			if errors.Is(err, syscall.EINTR) {
				continue
			}
		}
		break
	}

	c := make([]Channel, 0)
	if n > 0 {
		for i := 0; i < n; i++ {
			ch := p.channelMap[int(p.eventList[i].Fd)]
			ch.SetRevent(ReactorEvent(p.eventList[i].Events))
			c = append(c, p.channelMap[int(p.eventList[i].Fd)])
		}
	}

	if n == len(p.eventList) {
		p.eventList = make([]syscall.EpollEvent, 2*len(p.eventList))
	}

	return c
}

func (p *poller) UpdateChannel(c Channel) {
	if c.GetIndex() < 0 {
		err := syscall.EpollCtl(p.epollFD, syscall.EPOLL_CTL_ADD, c.GetFD(), &syscall.EpollEvent{
			Events: uint32(c.GetEvent()),
			Fd:     int32(c.GetFD()),
		})
		if err != nil {
			panic(err)
		}

		p.channelMap[c.GetFD()] = c
		c.SetIndex(p.idxCounter)
		p.idxCounter++
	} else {
		err := syscall.EpollCtl(p.epollFD, syscall.EPOLL_CTL_MOD, c.GetFD(), &syscall.EpollEvent{
			Events: uint32(c.GetEvent()),
			Fd:     int32(c.GetFD()),
		})
		if err != nil {
			panic(err)
		}
	}
}

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
