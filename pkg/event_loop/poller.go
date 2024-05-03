package eventloop

import (
	"fmt"
	"unsafe"

	"github.com/godzie44/go-uring/uring"
)

type poller struct {
	// uring file descriptor
	uringFD *uring.Ring

	// key is fd, value is Channel
	channelMap map[int]Channel

	// for each channel, assign a index, the first index is 1
	idxCounter int

	// store userdatas, prevent gc
	userDatas map[*userData]struct{}
}

// create a new poller, poller contains a epollfd,
func NewPoller() Poller {
	ring, err := uring.New(4096)
	if err != nil {
		panic(err)
	}

	return &poller{
		uringFD:    ring,
		channelMap: make(map[int]Channel),
		idxCounter: 0,
		userDatas:  make(map[*userData]struct{}),
	}
}

type Poller interface {
	// wait on epoll_wait and returns the active Channels
	Poll() map[Channel]struct{}

	// UpdateChannel calls epoll_ctl
	RegisterChannel(Channel)

	// RemoveChannel removes a fd from epollfd
	RemoveChannel(Channel)

	// GetChannelCount get current epoll wait fd nums, may be used to implement load balance
	GetChannelCount() int
}

type userData struct {
	fd      int
	op      ReactorEvent // op is ReadableEvent or WritableEvent
	readBuf []byte       // for read
}

// wait on epoll_wait and returns the active Channels
func (p *poller) Poll() map[Channel]struct{} {
	inQueueNum := 0
	for _, chIface := range p.channelMap {
		ch := chIface.(*channel)
		if ch.IsReading() && !ch.IsReadPending() {
			buf := make([]byte, 1024)
			uData := &userData{
				fd:      ch.GetFD(),
				op:      ReadableEvent,
				readBuf: buf,
			}
			p.userDatas[uData] = struct{}{}

			if ch.isAccept {
				err := p.uringFD.QueueSQE(uring.Accept(uintptr(ch.GetFD()), 0), 0, uint64(uintptr(unsafe.Pointer(uData))))
				if err != nil {
					panic(err)
				}
			} else {
				err := p.uringFD.QueueSQE(uring.Read(uintptr(ch.GetFD()), buf, 0), 0, uint64(uintptr(unsafe.Pointer(uData))))
				if err != nil {
					fmt.Println(inQueueNum)
					panic(err)
				}
			}
			inQueueNum++
			ch.DisableRead()
			ch.EnableReadPending()
		}
		if ch.IsWriting() && !ch.IsWritePending() {
			uData := &userData{
				fd:      ch.GetFD(),
				op:      WritableEvent,
				readBuf: nil, // do not need read buf
			}
			p.userDatas[uData] = struct{}{}

			err := p.uringFD.QueueSQE(uring.Write(uintptr(ch.GetFD()), ch.tobeWrite, 0), 0, uint64(uintptr(unsafe.Pointer(uData))))
			if err != nil {
				fmt.Println(inQueueNum)
				panic(err)
			}
			inQueueNum++
			ch.DisableWrite()
			ch.EnableWritePending()
		}
		if inQueueNum >= 4094 {
			n, err := p.uringFD.Submit()
			if n != uint(inQueueNum) || err != nil {
				panic(fmt.Sprintf("unexpected n=%v err=%v", n, err.Error()))
			}
			inQueueNum = 0
		}
	}

	_, err := p.uringFD.Submit()
	if err != nil {
		panic(err)
	}

retry:
	_, err = p.uringFD.WaitCQEvents(1)
	if err != nil {
		goto retry
	}
	resultCQEvents := make([]*uring.CQEvent, 4096)
	n := p.uringFD.PeekCQEventBatch(resultCQEvents)

	// fill up active channels
	c := make(map[Channel]struct{})
	for i := 0; i < n; i++ {
		cqe := resultCQEvents[i]
		p.uringFD.SeenCQE(cqe)
		uData := (*userData)(unsafe.Pointer(uintptr(cqe.UserData)))
		ch, ok := p.channelMap[uData.fd].(*channel)
		if ok {
			switch uData.op {
			case ReadableEvent:
				ch.EnableReventRead(uData.readBuf, int(cqe.Res))
			case WritableEvent:
				ch.EnableReventWrite(int(cqe.Res))
			}
			c[ch] = struct{}{}
		}
		delete(p.userDatas, uData)
	}

	return c
}

// UpdateChannel calls epoll_ctl
func (p *poller) RegisterChannel(c Channel) {
	// if index == -1, it is a new channel, assign a new index for it
	if c.GetIndex() < 0 {
		// assign an index
		p.idxCounter++
		p.channelMap[c.GetFD()] = c
		c.SetIndex(p.idxCounter)
	} else {
		panic("re register, check channel")
	}
}

// RemoveChannel removes a fd from epollfd
func (p *poller) RemoveChannel(c Channel) {
	if c.GetIndex() < 0 {
		panic("remove non-exist channel")
	}

	delete(p.channelMap, c.GetFD())
}

func (p *poller) GetChannelCount() int {
	return len(p.channelMap)
}
