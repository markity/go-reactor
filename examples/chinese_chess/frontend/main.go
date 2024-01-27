package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	commpackets "github.com/markity/go-reactor/examples/chinese_chess/frontend/comm_packets"
	commsettings "github.com/markity/go-reactor/examples/chinese_chess/frontend/comm_settings"
	"github.com/markity/go-reactor/examples/chinese_chess/frontend/tools"

	interactive "github.com/markity/Interactive-Console"
)

const (
	StateNone     = 0
	StateMatching = 1
	StateGaming   = 2
	StateOver     = 3
)

func main() {
	// 连上服务端
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", commsettings.ServerListenIP, commsettings.ServerListenPort))
	if err != nil {
		fmt.Printf("failed to dial to server: %v\n", err)
		return
	}

	heartbeatLoseCount := 0
	state := StateNone
	myTurn := false

	errChan := make(chan error, 1)
	readFromConnChan := make(chan interface{})
	heartbeatChan := time.NewTicker(commsettings.HeartbeatInterval * time.Millisecond)

	// conn reader
	go func() {
		for {
			packetBytes, err := tools.ReadPacketBytesWith4BytesHeader(conn)
			if err != nil {
				errChan <- err
				return
			}

			readFromConnChan <- commpackets.ClientParse(packetBytes)
		}
	}()

	// 写入start matching的包, 开始匹配
	startMatchPacket := commpackets.PacketClientStartMatch{}
	startMatchPacketBytesWithHeader := tools.DoPackWith4BytesHeader(startMatchPacket.MustMarshalToBytes())
	_, err = conn.Write(startMatchPacketBytesWithHeader)
	if err != nil {
		fmt.Printf("network error: %v\n", err)
		return
	}

	var win *interactive.Win = interactive.Run(interactive.GetDefaultConfig())
	win.SendLineBack("matching...")

	for {
		select {
		case cmd := <-win.GetCmdChan():
			if strings.TrimSpace(cmd) == "quit" {
				win.Stop()
				return
			}
			if state == StateGaming {
				var fromX, fromY, toX, toY int
				_, err := fmt.Sscanf(cmd, "%d %d %d %d", &fromX, &fromY, &toX, &toY)
				if err != nil {
					win.PopBackLine()
					win.SendLineBack("invalid input")
				}

				movePacket := commpackets.PacketClientMove{FromX: fromX, FromY: fromY, ToX: toX, ToY: toY}
				movePacketBytesWithHeader := tools.DoPackWith4BytesHeader(movePacket.MustMarshalToBytes())
				_, err = conn.Write(movePacketBytesWithHeader)
				if err != nil {
					win.Stop()
					fmt.Printf("network error: %v\n", err)
					return
				}
			}
		case err := <-errChan:
			win.Stop()
			fmt.Printf("error happened: %v\n", err)
			return
		case packIface := <-readFromConnChan:
			switch packet := packIface.(type) {
			case *commpackets.PacketServerMatchedOK:
				if state != StateMatching {
					panic("protocol error")
				}
				state = StateGaming
				msg := ""
				if packet.Side == commpackets.GameSideRed {
					msg = "it is your turn, red one"
					myTurn = true
				} else {
					msg = "it is not your turn, you are purple one"
					myTurn = false
				}
				tools.DrawTable(*packet.Table, win, msg)
			case *commpackets.PacketServerMatching:
				state = StateMatching
			case *commpackets.PacketServerMoveResp:
				if !packet.OK {
					win.PopBackLine()
					win.SendLineBack(*packet.ErrMsgOnFailed)
					continue
				}
				tools.DrawTable(*packet.TableOnOK, win, "it is not your turn")
				myTurn = false
			case *commpackets.PacketServerGameOver:
				winner := ""
				if myTurn {
					winner = "you"
				} else {
					winner = "him"
				}
				state = StateOver
				tools.DrawTable(*packet.Table, win, "game over, winner is "+winner)
				time.Sleep(time.Second * 10)
				win.Stop()
				return
			case *commpackets.PacketHeartbeat:
				heartbeatLoseCount = 0
			case *commpackets.PacketServerRemoteLoseConnection:
				win.Stop()
				fmt.Println("remote player closed connection")
				return
			case *commpackets.PacketServerNotifyRemoteMove:
				myTurn = true
				tools.DrawTable(*packet.Table, win, "it is your turn")
			}
		case <-heartbeatChan.C:
			heartbeatLoseCount++
			if heartbeatLoseCount >= commsettings.MaxLoseHeartbeat {
				win.Stop()
				fmt.Println("network error: connection lost")
				return
			}
			heartbeatPacket := commpackets.PacketHeartbeat{}
			heartbeatPacketBytesWithHeader := tools.DoPackWith4BytesHeader(heartbeatPacket.MustMarshalToBytes())
			_, err := conn.Write(heartbeatPacketBytesWithHeader)
			if err != nil {
				win.Stop()
				fmt.Printf("network error: %v\n", err)
				return
			}
		}
	}
}
