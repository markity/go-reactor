package async_log

import (
	"fmt"
	"go-reactor/pkg/cond"
	"os"
	"sync"
	"syscall"
	"time"
)

type LoggerLevel int

const (
	DEBUG LoggerLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (le LoggerLevel) String() string {
	switch le {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	}
	return "UNKNOWN"
}

type logger struct {
	mu   sync.Mutex
	cond cond.Cond

	currentBuffer *buffer
	backupBuffers []*buffer
	fullBuffers   []*buffer

	// readonly variables, can share without lock
	file       *os.File
	bufferSize int64
	level      LoggerLevel
}

type Logger interface {
	Logf(level LoggerLevel, f string, args ...interface{})
	Metrics() (backup int, full int)
}

func NewLogger(level LoggerLevel, path string, backupBufferNums int, bufferSize int64) Logger {
	if backupBufferNums <= 0 || bufferSize <= 0 {
		panic("check your params")
	}

	backup := make([]*buffer, 0)
	for i := 0; i < backupBufferNums; i++ {
		backup = append(backup, newLogBuffer(bufferSize))
	}
	lo := &logger{
		currentBuffer: newLogBuffer(bufferSize),
		backupBuffers: backup,
		fullBuffers:   make([]*buffer, 0),
		bufferSize:    bufferSize,
	}
	lo.cond = *cond.NewCond(&lo.mu)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	lo.file = f

	// 开启背景协程异步刷入到磁盘
	go func() {
		for {
			lo.mu.Lock()
			if lo.currentBuffer.Empty() {
				lo.cond.WaitWithTimeout(time.Second * 3)
			}
			if !lo.currentBuffer.Empty() {
				lo.fullBuffers = append(lo.fullBuffers, lo.currentBuffer)
				if len(lo.backupBuffers) != 0 {
					lo.currentBuffer = lo.backupBuffers[len(lo.backupBuffers)-1]
					lo.backupBuffers = lo.backupBuffers[:len(lo.backupBuffers)-1]
				} else {
					lo.currentBuffer = newLogBuffer(lo.bufferSize)
				}
			}
			tobeWritten := lo.fullBuffers
			lo.fullBuffers = make([]*buffer, 0)
			lo.mu.Unlock()

			for _, v := range tobeWritten {
				f.Write(v.data)
				v.Reset()
			}
			lo.mu.Lock()
			lo.backupBuffers = append(lo.backupBuffers, tobeWritten...)
			lo.mu.Unlock()
		}
	}()
	return lo
}

func (lo *logger) Logf(level LoggerLevel, f string, args ...interface{}) {
	if lo.level > level {
		return
	}

	lo.mu.Lock()
	defer lo.mu.Unlock()

	tv := syscall.Timeval{}
	err := syscall.Gettimeofday(&tv)
	if err != nil {
		panic(err)
	}

	s := time.Unix(tv.Sec, 0).Format("2006-01-02 15:04:05") + " " + level.String() + " " + fmt.Sprintf(f, args...) + "\n"
	if len(s) > int(lo.bufferSize) {
		panic("log is too large")
	}

	ok := lo.currentBuffer.Append([]byte(s))
	if !ok {
		lo.fullBuffers = append(lo.fullBuffers, lo.currentBuffer)
		if len(lo.backupBuffers) != 0 {
			lo.currentBuffer = lo.backupBuffers[len(lo.backupBuffers)-1]
			lo.backupBuffers = lo.backupBuffers[:len(lo.backupBuffers)-1]
		} else {
			lo.currentBuffer = newLogBuffer(lo.bufferSize)
		}
		lo.currentBuffer.Append([]byte(s))
		lo.cond.Broadcast()
	}
}

func (lo *logger) Metrics() (backup int, full int) {
	lo.mu.Lock()
	backup = len(lo.backupBuffers)
	full = len(lo.fullBuffers)
	lo.mu.Unlock()
	return
}
