package async_log

type buffer struct {
	data []byte
}

func (buf *buffer) Empty() bool {
	return len(buf.data) == 0
}

func (buf *buffer) Reset() {
	buf.data = buf.data[0:0]
}

func (buf *buffer) Append(bs []byte) bool {
	if cap(buf.data) < len(buf.data)+len(bs) {
		return false
	}

	buf.data = append(buf.data, bs...)
	return true
}

func newLogBuffer(size int64) *buffer {
	return &buffer{
		data: make([]byte, 0, size),
	}
}
