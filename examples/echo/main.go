package main

import (
	"fmt"
	goreactor "go-reactor"
	"go-reactor/pkg/buffer"
	eventloop "go-reactor/pkg/event_loop"
)

func main() {
	loop := eventloop.NewEventLoop()
	server := goreactor.NewTCPServer(loop, "127.0.0.1:8000")
	server.SetConnectionCallback(func(t *goreactor.TCPConnection) {
		fmt.Println("a new connection join")
		t.SetDisConnectedCallback(func(t *goreactor.TCPConnection) {
			fmt.Println("a connection disconnected")
		})
	})
	server.SetMessageCallback(func(t *goreactor.TCPConnection, b buffer.Buffer) {
		t.Send([]byte(b.RetrieveAsString()))
	})
	server.Start()
	loop.Loop()
}
