package goreactor

import (
	"go-reactor/pkg/buffer"
	eventloop "go-reactor/pkg/event_loop"
	"net/netip"
)

type ConnectedCallbackFunc func(*TCPConnection)
type MessageCallbackFunc func(*TCPConnection, buffer.Buffer)

type tcpServer struct {
	loop eventloop.EventLoop

	acceptor *tcpAcceptor

	// only be used to prevent double start
	started bool

	nextConnId int

	connectedCallback ConnectedCallbackFunc
	msgCallback       MessageCallbackFunc

	connections map[int]*TCPConnection
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

	server.started = true

	return nil
}

func NewTCPServer(loop eventloop.EventLoop, addrPort string) *tcpServer {
	listenAt, err := netip.ParseAddrPort(addrPort)
	if err != nil {
		panic(err)
	}

	acceptor := newTCPAcceptor(loop, listenAt)
	server := &tcpServer{
		loop:              loop,
		acceptor:          acceptor,
		started:           false,
		nextConnId:        0,
		connectedCallback: defaultConnectedCallback,
		msgCallback:       defaultMessageCallback,
		connections:       make(map[int]*TCPConnection),
	}

	acceptor.SetNewConnectionCallback(func(socketfd int, peerAddr netip.AddrPort) {
		conn := newConnection(acceptor.loop, socketfd, peerAddr)
		conn.setConnectedCallback(server.connectedCallback)
		conn.setCloseCallback(func(t *TCPConnection) {
			delete(server.connections, t.socketChannel.GetFD())
		})
		conn.setMessageCallback(server.msgCallback)
		conn.establishConn()
		server.nextConnId++
		server.connections[server.nextConnId] = conn
	})

	return server
}

func defaultConnectedCallback(tc *TCPConnection) {
	// just do nothing
}

func defaultMessageCallback(tc *TCPConnection, buf buffer.Buffer) {
	buf.RetrieveAsString()
}
