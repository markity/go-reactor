package goreactor

import (
	"net/netip"
	"syscall"

	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

type newConnectionCallback func(socketfd int, peerAddr netip.AddrPort)

type tcpAcceptor struct {
	// event loop
	loop eventloop.EventLoop

	// be used to prevent double start
	listening bool

	// listen at
	listenAddr netip.AddrPort

	// listen socket fd channel
	socketChannel eventloop.Channel

	// new connection call back
	newConnectionCallback newConnectionCallback

	// syscall.Listen param, see man 2 listen()
	// The backlog argument defines the maximum length to which the queue of pending connections for sockfd may grow.  If a  connection  request  arrives  when  the
	// queue  is full, the client may receive an error with an indication of ECONNREFUSED or, if the underlying protocol supports retransmission, the request may be
	// ignored so that a later reattempt at connection succeeds.
	listenBackup int
}

func newTCPAcceptor(loop eventloop.EventLoop, listenAddr netip.AddrPort, listenBackup int) *tcpAcceptor {
	socketFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}

	c := eventloop.NewChannel(socketFD)

	acc := tcpAcceptor{
		loop:                  loop,
		listenAddr:            listenAddr,
		listening:             false,
		socketChannel:         c,
		newConnectionCallback: defaultNewConnectionCallback,
		listenBackup:          listenBackup,
	}
	c.SetReadCallback(acc.HandleRead)

	// 15 means SO_REUSEPORT
	err = syscall.SetsockoptInt(socketFD, syscall.SOL_SOCKET, 15, 1)
	if err != nil {
		panic(err)
	}

	return &acc
}

func (ac *tcpAcceptor) Listen() error {
	if ac.listening {
		panic("already listening")
	}

	err := syscall.Bind(ac.socketChannel.GetFD(), &syscall.SockaddrInet4{
		Addr: ac.listenAddr.Addr().As4(),
		Port: int(ac.listenAddr.Port()),
	})
	if err != nil {
		return err
	}

	err = syscall.Listen(ac.socketChannel.GetFD(), ac.listenBackup)
	if err != nil {
		return err
	}

	ac.listening = true
	ac.socketChannel.EnableRead(true)
	ac.loop.RegisterChannelInLoopGoroutine(ac.socketChannel)
	return nil
}

func (ac *tcpAcceptor) HandleRead(bs []byte, res int) {
	ac.socketChannel.DisableReadPending()

	if ac.newConnectionCallback != nil {
		ac.newConnectionCallback(res, netip.AddrPort{})
	}

	ac.socketChannel.EnableRead(true)
}

func (ac *tcpAcceptor) SetNewConnectionCallback(cb newConnectionCallback) {
	ac.newConnectionCallback = cb
}

func defaultNewConnectionCallback(socketfd int, peerAddr netip.AddrPort) {
	syscall.Close(socketfd)
}
