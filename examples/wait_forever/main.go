package main

import (
	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

func main() {
	loop := eventloop.NewEventLoop()
	loop.Loop()
}
