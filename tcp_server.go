package goreactor

import (
	"net/netip"

	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

type TCPServer interface {
	SetConnectionCallback(f ConnectedCallbackFunc)
	SetMessageCallback(f MessageCallbackFunc)
	Start() error
}

type tcpServer struct {
	loop eventloop.EventLoop

	acceptor *tcpAcceptor

	// only be used to prevent double start
	started bool

	connectedCallback ConnectedCallbackFunc
	msgCallback       MessageCallbackFunc

	evloopPoll *eventloopGoroutinePoll

	loadBalanceStrategy LoadBalanceStrategy
}

func (server *tcpServer) SetConnectionCallback(f ConnectedCallbackFunc) {
	server.connectedCallback = f
}

func (server *tcpServer) SetMessageCallback(f MessageCallbackFunc) {
	server.msgCallback = f
}

func (server *tcpServer) Start() error {
	if server.started {
		panic("already started")
	}

	err := server.acceptor.Listen()
	if err != nil {
		return err
	}

	server.evloopPoll.start()

	server.started = true

	return nil
}

func (server *tcpServer) onNewConnection(socketfd int, peerAddr netip.AddrPort) {
	loop := server.evloopPoll.getNext()

	conn := newConnection(loop, socketfd, peerAddr)
	conn.setConnectedCallback(server.connectedCallback)
	conn.setCloseCallback(func(t TCPConnection) {
		// just do nothing
	})
	conn.setMessageCallback(server.msgCallback)

	loop.RunInLoop(func() {
		conn.establishConn()
	})

}

// if numWorkingThread is 0, all channel will run on single loop
func NewTCPServer(loop eventloop.EventLoop, addrPort string,
	numWorkingThread int, strategy LoadBalanceStrategy) TCPServer {
	listenAt, err := netip.ParseAddrPort(addrPort)
	if err != nil {
		panic(err)
	}

	if numWorkingThread < 0 {
		panic(numWorkingThread)
	}

	acceptor := newTCPAcceptor(loop, listenAt, 1024)
	server := &tcpServer{
		loop:                loop,
		acceptor:            acceptor,
		started:             false,
		connectedCallback:   defaultConnectedCallback,
		msgCallback:         defaultMessageCallback,
		evloopPoll:          newEventloopGoroutinePoll(loop, numWorkingThread, strategy),
		loadBalanceStrategy: strategy,
	}

	acceptor.SetNewConnectionCallback(server.onNewConnection)

	return server
}
