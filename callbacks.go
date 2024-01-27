package goreactor

import "github.com/markity/go-reactor/pkg/buffer"

type ConnectedCallbackFunc func(TCPConnection)
type DisConnectedCallbackFunc func(TCPConnection)
type MessageCallbackFunc func(TCPConnection, buffer.Buffer)
type HighWaterCallbackFunc func(TCPConnection, int)
type WriteCompleteCallbackFunc func(TCPConnection)

func defaultHighWaterMarkCallback(tc TCPConnection, sz int) {
	// just do nothing
}

func defaultWriteCompleteCallback(tc TCPConnection) {
	// just do nothing
}

func defaultDisConnectedCallback(tc TCPConnection) {
	// just do nothing
}

func defaultConnectedCallback(tc TCPConnection) {
	// just do nothing
}

func defaultMessageCallback(tc TCPConnection, buf buffer.Buffer) {
	buf.RetrieveAsString()
}
