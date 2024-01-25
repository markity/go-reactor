package goreactor

import (
	"go-reactor/pkg/buffer"
	eventloop "go-reactor/pkg/event_loop"
	"log"
	"net/netip"
	"syscall"
)

type tcpConnectionState int

const (
	Connecting    tcpConnectionState = 1
	Connected     tcpConnectionState = 2
	Disconnecting tcpConnectionState = 3
	Disconnected  tcpConnectionState = 4
)

type closeCallbackFunc func(*TCPConnection)
type HighWaterCallbackFunc func(*TCPConnection, int)
type WriteCompleteCallbackFunc func(*TCPConnection)
type DisConnectedCallbackFunc func(*TCPConnection)

// 能被多个协程share
type TCPConnection struct {
	state tcpConnectionState

	loop eventloop.EventLoop

	socketChannel eventloop.Channel

	connectedCallback     ConnectedCallbackFunc
	disconnectedCallback  DisConnectedCallbackFunc
	closeCallback         closeCallbackFunc
	messageCallback       MessageCallbackFunc
	highWaterCallback     HighWaterCallbackFunc
	writeCompleteCallback WriteCompleteCallbackFunc

	// 0 means infinite
	hignWaterLevel int

	remoteAddrPort netip.AddrPort

	outputBuffer buffer.Buffer
	inputBuffer  buffer.Buffer
}

func (tc *TCPConnection) setConnectedCallback(f ConnectedCallbackFunc) {
	tc.connectedCallback = f
}

func (tc *TCPConnection) setDisConnectedCallback(f DisConnectedCallbackFunc) {
	tc.disconnectedCallback = f
}

func (tc *TCPConnection) setCloseCallback(f closeCallbackFunc) {
	tc.closeCallback = f
}

func (tc *TCPConnection) setMessageCallback(f MessageCallbackFunc) {
	tc.messageCallback = f
}

func (tc *TCPConnection) SetHighWaterCallback(f HighWaterCallbackFunc) {
	tc.highWaterCallback = f
}

// 0 means infinite
func (tc *TCPConnection) SetHighWaterLevel(i int) {
	tc.hignWaterLevel = i
}

func (tc *TCPConnection) SetWriteCompleteCallback(f WriteCompleteCallbackFunc) {
	tc.writeCompleteCallback = f
}

func (tc *TCPConnection) SetDisConnectedCallback(f DisConnectedCallbackFunc) {
	tc.disconnectedCallback = f
}

func newConnection(loop eventloop.EventLoop, sockFD int, remoteAddrPort netip.AddrPort) *TCPConnection {
	channel := eventloop.NewChannel(sockFD)
	c := &TCPConnection{
		state:                 Connecting,
		loop:                  loop,
		socketChannel:         channel,
		outputBuffer:          buffer.NewBuffer(),
		inputBuffer:           buffer.NewBuffer(),
		remoteAddrPort:        remoteAddrPort,
		highWaterCallback:     defaultHighWaterMarkCallback,
		writeCompleteCallback: defaultWriteCompleteCallback,
		disconnectedCallback:  defaultDisConnectedCallback,
	}
	channel.SetCloseCallback(c.handleClose)
	channel.SetErrorCallback(c.handleError)
	channel.SetReadCallback(c.handleRead)
	channel.SetWriteCallback(c.handleWrite)
	channel.SetEvent(eventloop.CloseEvent | eventloop.ErrorEvent | eventloop.ReadableEvent | eventloop.WritableEvent)

	return c
}

func (conn *TCPConnection) Send(bs []byte) {
	copyBs := make([]byte, len(bs))
	copy(copyBs, bs)
	conn.loop.RunInLoop(func() {
		if conn.state == Connected {
			conn.outputBuffer.Append(copyBs)
			if conn.hignWaterLevel != 0 && conn.outputBuffer.ReadableBytes() > conn.hignWaterLevel {
				conn.highWaterCallback(conn, conn.outputBuffer.ReadableBytes())
			}
			if !conn.socketChannel.IsWriting() {
				conn.socketChannel.EnableWrite()
				conn.loop.UpdateChannelInLoopGoroutine(conn.socketChannel)
			}
		}
	})
}

func (conn *TCPConnection) Shutdown() {
	conn.loop.RunInLoop(func() {
		if conn.state == Connected {
			conn.state = Disconnecting
			syscall.Shutdown(conn.socketChannel.GetFD(), syscall.SHUT_WR)
		}
	})
}

func (conn *TCPConnection) GetRemoteAddrPort() netip.AddrPort {
	return conn.remoteAddrPort
}

func (conn *TCPConnection) ForceClose() {
	conn.loop.RunInLoop(func() {
		if conn.state == Disconnecting || conn.state == Connected {
			conn.handleClose()
		}
	})
}

func (conn *TCPConnection) SetKeepAlive(b bool) {
	conn.loop.RunInLoop(func() {
		val := 0
		if b {
			val = 1
		}
		syscall.SetsockoptInt(conn.socketChannel.GetFD(), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, val)
	})
}

func (conn *TCPConnection) SetNoDelay(b bool) {
	conn.loop.RunInLoop(func() {
		val := 0
		if b {
			val = 1
		}
		syscall.SetsockoptInt(conn.socketChannel.GetFD(), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, val)
	})
}

func (conn *TCPConnection) handleRead() {
	n := conn.inputBuffer.ReadFD(conn.socketChannel.GetFD())
	if n > 0 {
		conn.messageCallback(conn, conn.inputBuffer)
	} else if n == 0 {
		// n为0意味对面已经close write或close total了, 此时直接关闭连接
		conn.handleClose()
	} else {
		panic("checkme")
	}
}

func (conn *TCPConnection) handleWrite() {
	n, err := syscall.Write(conn.socketChannel.GetFD(), conn.outputBuffer.Peek()[:conn.outputBuffer.ReadableBytes()])
	if err != nil {
		log.Printf("syscall.Write socket: %v\n", err)
	}
	conn.outputBuffer.Retrieve(n)
	if conn.outputBuffer.ReadableBytes() == 0 {
		conn.socketChannel.DisableWrite()
		conn.loop.UpdateChannelInLoopGoroutine(conn.socketChannel)

		conn.writeCompleteCallback(conn)
	}
}

func (conn *TCPConnection) handleError() {
	log.Println("handle error")
}

// 比如对端直接close了socket, 那么此时就进入readhup状态了
func (conn *TCPConnection) handleClose() {
	if conn.state != Disconnecting && conn.state != Connected {
		panic("checkme")
	}

	conn.state = Disconnected
	conn.loop.RemoveChannelInLoopGoroutine(conn.socketChannel)
	syscall.Close(conn.socketChannel.GetFD())
	conn.closeCallback(conn)
	conn.disconnectedCallback(conn)
}

func (conn *TCPConnection) establishConn() {
	if conn.state != Connecting {
		panic("unexpected")
	}

	conn.state = Connected
	conn.loop.UpdateChannelInLoopGoroutine(conn.socketChannel)
	conn.connectedCallback(conn)
}

func defaultHighWaterMarkCallback(tc *TCPConnection, sz int) {
	// just do nothing
}

func defaultWriteCompleteCallback(tc *TCPConnection) {
	// just do nothing
}

func defaultDisConnectedCallback(tc *TCPConnection) {
	// just do nothing
}
