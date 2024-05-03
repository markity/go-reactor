package buffer

import (
	"syscall"
	"unsafe"
)

type buffer struct {
	data       []byte
	readIndex  int
	writeIndex int
}

func (buf *buffer) ReadableBytes() int {
	return buf.writeIndex - buf.readIndex
}

func (buf *buffer) Peek() []byte {
	return buf.data[buf.readIndex:buf.writeIndex]
}

func (buf *buffer) Retrieve(i int) {
	if buf.ReadableBytes() < i {
		panic("retrieve too many bytes")
	}

	buf.readIndex += i
}

func (buf *buffer) RetrieveAll() {
	buf.readIndex = 0
	buf.writeIndex = 0
}

func (buf *buffer) RetrieveAsString() string {
	s := string(buf.data[buf.readIndex:buf.writeIndex])
	buf.readIndex = 0
	buf.writeIndex = 0
	return s
}

func (buf *buffer) Append(bs []byte) {
	if len(buf.data)-buf.writeIndex >= len(bs) {
		copy(buf.data[buf.writeIndex:], bs)
		buf.writeIndex += len(bs)
	} else {
		if len(buf.data) >= buf.writeIndex-buf.readIndex+len(bs) {
			sz := buf.writeIndex - buf.readIndex + len(bs)
			copy(buf.data, buf.data[buf.readIndex:buf.readIndex+buf.writeIndex-buf.readIndex])
			copy(buf.data[buf.writeIndex-buf.readIndex:], bs)
			buf.readIndex = 0
			buf.writeIndex = sz
		} else {
			newBytes := make([]byte, buf.writeIndex-buf.readIndex+len(bs)+8192)
			sz := buf.writeIndex - buf.readIndex + len(bs)
			copy(newBytes, buf.data[buf.readIndex:buf.writeIndex])
			copy(newBytes[buf.writeIndex:], bs)
			buf.data = newBytes
			buf.readIndex = 0
			buf.writeIndex = sz
		}
	}
}

func (buf *buffer) ReadFD(fd int, extrabuf []byte) int {
	writable := len(buf.data) - buf.writeIndex
	ptr := uintptr(unsafe.Pointer(&buf.data[buf.writeIndex]))
	base := (*byte)(unsafe.Pointer(ptr))
	iovec := [2]syscall.Iovec{
		{
			Base: base,
			Len:  uint64(writable),
		},
		{
			Base: (*byte)(unsafe.Pointer(&extrabuf[0])),
			Len:  uint64(len(extrabuf)),
		},
	}

	sz, _, _ := syscall.Syscall(syscall.SYS_READV, uintptr(fd), uintptr(unsafe.Pointer(&iovec)), 2)

	size := int(sz)
	if size < 0 {
		size = 0
	}

	if size == 0 {
		return 0
	}

	if size <= writable {
		buf.writeIndex += size
	} else {
		buf.writeIndex = len(buf.data)
		buf.Append(extrabuf[:size-writable])
	}

	return size
}

func NewBuffer() Buffer {
	return &buffer{
		data:       make([]byte, 8192),
		readIndex:  0,
		writeIndex: 0,
	}
}

type Buffer interface {
	ReadableBytes() int
	Peek() []byte
	Retrieve(int)
	RetrieveAll()
	RetrieveAsString() string
	Append([]byte)
	ReadFD(int, []byte) int
}
