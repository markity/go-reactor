package main

import (
	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

func main() {
	loop := eventloop.NewEventLoop(0)
	loop.Loop()
}
