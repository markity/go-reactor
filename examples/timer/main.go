package main

import (
	"fmt"
	"time"

	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

var i int

func onTimeout(loop eventloop.EventLoop, timerId int) {
	i++
	fmt.Println("timeout", timerId)
	if i == 3 {
		loop.CancelTimer(timerId)
		loop.RunAt(time.Now(), time.Second*1, func(loop eventloop.EventLoop, timerID int) {
			fmt.Println("timeout 2")
		})
	}
}

func main() {
	loop := eventloop.NewEventLoop()
	loop.RunAt(time.Now(), time.Second*3, onTimeout)
	loop.Loop()
}
