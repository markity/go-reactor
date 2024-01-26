package main

import (
	"fmt"
	goreactor "go-reactor"
	"go-reactor/pkg/buffer"
	eventloop "go-reactor/pkg/event_loop"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		if err := http.ListenAndServe(":6089", nil); err != nil {
			panic(err)
		}
	}()

	evloop := eventloop.NewEventLoop(0)
	server := goreactor.NewTCPServer(evloop, "127.0.0.1:8000", 8, goreactor.RoundRobin())
	server.SetConnectionCallback(func(t goreactor.TCPConnection) {
		t.SetDisConnectedCallback(func(t goreactor.TCPConnection) {
			fmt.Println("connection disconnects")
		})
		fmt.Println("a new connection comes up, loop is", t.GetEventLoop().GetID())
	})
	server.SetMessageCallback(func(t goreactor.TCPConnection, b buffer.Buffer) {
		t.Send([]byte(b.RetrieveAsString()))
	})
	server.Start()
	evloop.Loop()
}
