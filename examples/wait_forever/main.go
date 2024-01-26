package main

import (
	eventloop "go-reactor/pkg/event_loop"
)

func main() {
	loop := eventloop.NewEventLoop(0)
	loop.Loop()
}
