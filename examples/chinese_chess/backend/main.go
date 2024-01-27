package main

import (
	"fmt"
	"time"

	goreactor "github.com/markity/go-reactor"
	commsettings "github.com/markity/go-reactor/examples/chinese_chess/backend/common_settings"
	gamehandler "github.com/markity/go-reactor/examples/chinese_chess/backend/game_handler"
	eventloop "github.com/markity/go-reactor/pkg/event_loop"
)

func main() {
	loop := eventloop.NewEventLoop(0)
	loop.RunAt(time.Now(), time.Millisecond*commsettings.HeartbeatInterval, gamehandler.OnTimeout)

	listenIPPort := fmt.Sprintf("%v:%v", commsettings.ServerListenIP, commsettings.ServerListenPort)
	fmt.Println(listenIPPort)
	server := goreactor.NewTCPServer(loop, listenIPPort, 0, goreactor.RoundRobin())
	server.SetConnectionCallback(gamehandler.OnConnect)
	server.SetMessageCallback(gamehandler.OnMessage)

	err := server.Start()
	if err != nil {
		panic(err)
	}
	loop.Loop()
}
