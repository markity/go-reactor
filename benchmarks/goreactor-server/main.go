package main

import (
	eventloop "github.com/markity/go-reactor/pkg/event_loop"

	"github.com/markity/go-reactor/pkg/buffer"

	goreactor "github.com/markity/go-reactor"
)

func main() {
	evloop := eventloop.NewEventLoop()
	server := goreactor.NewTCPServer(evloop, "127.0.0.1:8000", 0, goreactor.RoundRobin())
	server.SetMessageCallback(func(t goreactor.TCPConnection, b buffer.Buffer) {
		t.Send(b.Peek())
		b.RetrieveAll()
	})
	server.Start()
	evloop.Loop()
}
