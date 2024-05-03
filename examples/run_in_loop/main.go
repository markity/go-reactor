package main

import (
	"fmt"
	"time"

	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

func main() {
	loop := eventloop.NewEventLoop()
	go func() {
		for {
			loop.RunInLoop(func() {
				fmt.Println("test")
			})
			time.Sleep(time.Second)
		}
	}()
	loop.Loop()
}
