package main

import (
	goreactor "go-reactor"
	"go-reactor/pkg/buffer"
	eventloop "go-reactor/pkg/event_loop"
)

func main() {
	evloop := eventloop.NewEventLoop(0)
	server := goreactor.NewTCPServer(evloop, "127.0.0.1:8000", 0, goreactor.RoundRobin())
	server.SetMessageCallback(func(t goreactor.TCPConnection, b buffer.Buffer) {
		t.Send(b.Peek())
		b.RetrieveAll()
	})
	server.Start()
	evloop.Loop()
}
