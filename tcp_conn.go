package goreactor

import (
	"net/netip"
	"syscall"

	kvcontext "github.com/markity/go-reactor/pkg/context"
	eventloop "github.com/markity/go-reactor/pkg/event_loop"

	"github.com/markity/go-reactor/pkg/buffer"
)

type tcpConnectionState int

const (
	Connecting    tcpConnectionState = 1
	Connected     tcpConnectionState = 2
	Disconnecting tcpConnectionState = 3
	Disconnected  tcpConnectionState = 4
)

type TCPConnection interface {
	SetDisConnectedCallback(f DisConnectedCallbackFunc)
	SetHighWaterCallback(f HighWaterCallbackFunc)
	SetWriteCompleteCallback(f WriteCompleteCallbackFunc)
	Send(bs []byte)
	ShutdownWrite()
	GetRemoteAddrPort() netip.AddrPort
	ForceClose()
	SetKeepAlive(b bool)
	SetNoDelay(b bool)
	GetEventLoop() eventloop.EventLoop
	SetContext(key string, value interface{})
	DeleteContext(key string)
	GetContext(key string) (interface{}, bool)
	MustGetContext(key string) interface{}
	GetFD() int
	IsConnected() bool
}

type closeCallbackFunc func(TCPConnection)

// 能被多个协程share
type tcpConnection struct {
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

	ctx kvcontext.KVContext
}

func (tc *tcpConnection) setConnectedCallback(f ConnectedCallbackFunc) {
	tc.connectedCallback = f
}

func (tc *tcpConnection) setCloseCallback(f closeCallbackFunc) {
	tc.closeCallback = f
}

func (tc *tcpConnection) setMessageCallback(f MessageCallbackFunc) {
	tc.messageCallback = f
}

func (tc *tcpConnection) SetHighWaterCallback(f HighWaterCallbackFunc) {
	tc.highWaterCallback = f
}

// 0 means infinite
func (tc *tcpConnection) SetHighWaterLevel(i int) {
	tc.hignWaterLevel = i
}

func (tc *tcpConnection) SetWriteCompleteCallback(f WriteCompleteCallbackFunc) {
	tc.writeCompleteCallback = f
}

func (tc *tcpConnection) SetDisConnectedCallback(f DisConnectedCallbackFunc) {
	tc.disconnectedCallback = f
}

func newConnection(loop eventloop.EventLoop, sockFD int, remoteAddrPort netip.AddrPort) *tcpConnection {
	channel := eventloop.NewChannel(sockFD)
	c := &tcpConnection{
		state:                 Connecting,
		loop:                  loop,
		socketChannel:         channel,
		outputBuffer:          buffer.NewBuffer(),
		inputBuffer:           buffer.NewBuffer(),
		remoteAddrPort:        remoteAddrPort,
		highWaterCallback:     defaultHighWaterMarkCallback,
		writeCompleteCallback: defaultWriteCompleteCallback,
		disconnectedCallback:  defaultDisConnectedCallback,
		ctx:                   kvcontext.NewContext(),
	}
	channel.SetReadCallback(c.handleRead)
	channel.SetWriteCallback(c.handleWrite)
	channel.SetEvent(eventloop.ReadableEvent | eventloop.WritableEvent)

	return c
}

func (conn *tcpConnection) Send(bs []byte) {
	conn.loop.RunInLoop(func() {
		if conn.state == Connected {
			conn.outputBuffer.Append(bs)
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

func (conn *tcpConnection) ShutdownWrite() {
	conn.loop.RunInLoop(func() {
		if conn.state == Connected {
			conn.state = Disconnecting
			if !conn.socketChannel.IsWriting() {
				syscall.Shutdown(conn.socketChannel.GetFD(), syscall.SHUT_WR)
			}
		}
	})
}

func (conn *tcpConnection) GetRemoteAddrPort() netip.AddrPort {
	return conn.remoteAddrPort
}

func (conn *tcpConnection) ForceClose() {
	conn.loop.RunInLoop(func() {
		if conn.state == Disconnecting || conn.state == Connected {
			conn.handleClose()
		}
	})
}

func (conn *tcpConnection) SetKeepAlive(b bool) {
	conn.loop.RunInLoop(func() {
		val := 0
		if b {
			val = 1
		}
		syscall.SetsockoptInt(conn.socketChannel.GetFD(), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, val)
	})
}

func (conn *tcpConnection) SetNoDelay(b bool) {
	conn.loop.RunInLoop(func() {
		val := 0
		if b {
			val = 1
		}
		syscall.SetsockoptInt(conn.socketChannel.GetFD(), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, val)
	})
}

func (conn *tcpConnection) handleRead() {
	n := conn.inputBuffer.ReadFD(conn.socketChannel.GetFD(), conn.GetEventLoop().GetExtraData())
	if n > 0 {
		conn.messageCallback(conn, conn.inputBuffer)
	} else if n <= 0 {
		// n为0意味对面已经close write或close total了, 此时直接关闭连接
		conn.handleClose()
	}
}

func (conn *tcpConnection) handleWrite() {
	n, _ := syscall.Write(conn.socketChannel.GetFD(), conn.outputBuffer.Peek()[:conn.outputBuffer.ReadableBytes()])
	conn.outputBuffer.Retrieve(n)
	if conn.outputBuffer.ReadableBytes() == 0 {
		conn.socketChannel.DisableWrite()
		conn.loop.UpdateChannelInLoopGoroutine(conn.socketChannel)

		conn.writeCompleteCallback(conn)
		if conn.state == Disconnecting {
			syscall.Shutdown(conn.socketChannel.GetFD(), syscall.SHUT_WR)
		}
	}
}

// 比如对端直接close了socket, 那么此时就进入readhup状态了
func (conn *tcpConnection) handleClose() {
	if conn.state != Disconnecting && conn.state != Connected {
		panic("checkme")
	}

	conn.state = Disconnected
	conn.loop.RemoveChannelInLoopGoroutine(conn.socketChannel)
	syscall.Close(conn.socketChannel.GetFD())
	conn.closeCallback(conn)
	conn.disconnectedCallback(conn)
}

func (conn *tcpConnection) establishConn() {
	if conn.state != Connecting {
		panic("unexpected")
	}

	conn.state = Connected
	conn.loop.UpdateChannelInLoopGoroutine(conn.socketChannel)
	conn.connectedCallback(conn)
}

func (conn *tcpConnection) GetEventLoop() eventloop.EventLoop {
	return conn.loop
}

func (conn *tcpConnection) SetContext(key string, value interface{}) {
	conn.ctx.Set(key, value)
}

func (conn *tcpConnection) DeleteContext(key string) {
	conn.ctx.Delete(key)
}

func (conn *tcpConnection) GetContext(key string) (interface{}, bool) {
	return conn.ctx.Get(key)
}

func (ev *tcpConnection) MustGetContext(key string) interface{} {
	v, ok := ev.ctx.Get(key)
	if !ok {
		panic("must get context not exists")
	}
	return v
}

func (conn *tcpConnection) GetFD() int {
	return conn.socketChannel.GetFD()
}

func (conn *tcpConnection) IsConnected() bool {
	c := make(chan bool, 1)
	conn.loop.RunInLoop(func() {
		c <- conn.state == Connected
	})
	return <-c
}
