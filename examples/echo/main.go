package main

import (
	"fmt"

	eventloop "github.com/markity/go-reactor/pkg/event_loop"

	"github.com/markity/go-reactor/pkg/buffer"

	goreactor "github.com/markity/go-reactor"
)

func main() {
	loop := eventloop.NewEventLoop(0)

	server := goreactor.NewTCPServer(loop, "127.0.0.1:8000", 0, goreactor.RoundRobin())
	server.SetConnectionCallback(func(t goreactor.TCPConnection) {
		fmt.Println("a new connection join, loop is", t.GetEventLoop().GetID())
		t.SetDisConnectedCallback(func(t goreactor.TCPConnection) {
			fmt.Println("a connection disconnected")
		})
	})
	server.SetMessageCallback(func(t goreactor.TCPConnection, b buffer.Buffer) {
		t.Send([]byte(b.RetrieveAsString()))
	})
	server.Start()
	loop.Loop()
}
