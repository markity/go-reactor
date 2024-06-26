package main

import (
	"fmt"
	"time"

	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

func onTimeout(timerId int) {
	fmt.Println("timeout", timerId)
}

func main() {
	loop := eventloop.NewEventLoop()
	loop.RunAt(time.Now(), time.Second*3, onTimeout)
	loop.Loop()
}
