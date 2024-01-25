package eventloop

import (
	"container/heap"
	"syscall"
	"time"
	"unsafe"
)

// be used for timerfd_settime
type itimerspec struct {
	it_interval syscall.Timespec
	it_value    syscall.Timespec
}

// 为了区分一个时间戳不同
type timerHeapEntry struct {
	timerId   int
	TimeStamp time.Time
	onTimer   func()
	interval  time.Duration
}

type timerHeap []timerHeapEntry

func (th *timerHeap) Len() int {
	return len(*th)
}

func (th *timerHeap) Less(i, j int) bool {
	if (*th)[i].TimeStamp.Equal((*th)[j].TimeStamp) {
		return (*th)[i].timerId < (*th)[j].timerId
	}

	return (*th)[i].TimeStamp.Before((*th)[j].TimeStamp)
}

func (th *timerHeap) Swap(i, j int) {
	(*th)[i], (*th)[j] = (*th)[j], (*th)[i]
}

func (th *timerHeap) Push(x interface{}) {
	*th = append(*th, x.(timerHeapEntry))
}

func (th *timerHeap) Pop() interface{} {
	old := *th
	n := len(old)
	x := old[n-1]
	*th = old[0 : n-1]
	return x
}

type timerQueue struct {
	timerChannel   Channel
	timerFD        int
	timerIdCounter int
	heap           timerHeap
	loop           EventLoop
}

func newTimerQueue(loop EventLoop) *timerQueue {
	// 1 means CLOCK_MONOTONIC, see timerfd_create(2)
	timerfd, _, errno := syscall.Syscall(syscall.SYS_TIMERFD_CREATE, 1, syscall.O_NONBLOCK, 0)
	if errno != 0 {
		panic(errno)
	}

	ch := NewChannel(int(timerfd))
	ch.SetEvent(ReadableEvent)
	tq := timerQueue{
		timerFD:        int(timerfd),
		timerChannel:   ch,
		timerIdCounter: 0,
		heap:           make(timerHeap, 0),
		loop:           loop,
	}
	heap.Init(&tq.heap)
	loop.UpdateChannelInLoopGoroutine(ch)

	ch.SetReadCallback(func() {
		_, err := syscall.Read(int(timerfd), make([]byte, 8))
		if err != nil {
			panic(err)
		}

		for _, v := range tq.getExpired() {
			v.onTimer()
		}
	})

	return &tq
}

func (tq *timerQueue) AddTimer(triggerAt time.Time, interval time.Duration, f func()) int {
	id := tq.timerIdCounter
	tq.timerIdCounter++
	heap.Push(&tq.heap, timerHeapEntry{
		timerId:   id,
		TimeStamp: triggerAt,
		onTimer:   f,
		interval:  interval,
	})

	nsec := 0

	now := time.Now()
	earliest := tq.heap[0].TimeStamp
	if now.Before(earliest) {
		interval := earliest.Sub(now)
		nsec = int(interval.Nanoseconds())
	}

	sp := itimerspec{
		it_value: syscall.Timespec{
			Sec:  int64(nsec / 1000000000),
			Nsec: int64(nsec % 1000000000),
		},
		// set both to zero means trigger only once
		it_interval: syscall.Timespec{Sec: 0, Nsec: 0},
	}

	// 1 means TFD_TIMER_ABSTIME, see timerfd_setime(2)
	_, _, errno := syscall.Syscall6(syscall.SYS_TIMERFD_SETTIME, uintptr(tq.timerFD), 2, uintptr(unsafe.Pointer(&sp)), 0, 0, 0)
	if errno != 0 {
		panic(errno)
	}

	return id
}

func (tq *timerQueue) CancelTimer(timerId int) bool {
	newHeap := make([]timerHeapEntry, 0)
	ok := false
	for i, v := range tq.heap {
		if v.timerId == timerId {
			ok = true
			continue
		}
		newHeap = append(newHeap, tq.heap[i])
	}
	tq.heap = newHeap
	return ok
}

func (tq *timerQueue) getExpired() []timerHeapEntry {
	te := make([]timerHeapEntry, 0)
	now := time.Now()
	for {
		if tq.heap.Len() == 0 {
			break
		}

		minOne := tq.heap[0]
		if minOne.TimeStamp.Before(now) {
			te = append(te, tq.heap[0])
			heap.Pop(&tq.heap)
			if minOne.interval != 0 {
				minOne.TimeStamp = minOne.TimeStamp.Add(minOne.interval)
				heap.Push(&tq.heap, minOne)
			}
		} else {
			break
		}
	}

	if tq.heap.Len() != 0 {
		nsec := 0

		now := time.Now()
		earliest := tq.heap[0].TimeStamp
		if now.Before(earliest) {
			interval := earliest.Sub(now)
			nsec = int(interval.Nanoseconds())
		}

		sp := itimerspec{
			it_value: syscall.Timespec{
				Sec:  int64(nsec / 1000000000),
				Nsec: int64(nsec % 1000000000),
			},
			// set both to zero means trigger only once
			it_interval: syscall.Timespec{Sec: 0, Nsec: 0},
		}

		// 1 means TFD_TIMER_ABSTIME, see timerfd_setime(2)
		_, _, errno := syscall.Syscall6(syscall.SYS_TIMERFD_SETTIME, uintptr(tq.timerFD), 2, uintptr(unsafe.Pointer(&sp)), 0, 0, 0)
		if errno != 0 {
			panic(errno)
		}
	}
	return te
}
