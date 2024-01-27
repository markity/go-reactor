package gamehandler

import (
	"fmt"

	goreactor "github.com/markity/go-reactor"
	commsettings "github.com/markity/go-reactor/examples/chinese_chess/backend/common_settings"
	"github.com/markity/go-reactor/examples/chinese_chess/backend/tools"
	"github.com/markity/go-reactor/pkg/buffer"

	commpackets "github.com/markity/go-reactor/examples/chinese_chess/backend/common_packets"

	"github.com/Allenxuxu/toolkit/sync/atomic"
)

type ConnState int

const (
	ConnStateNone ConnState = iota
	ConnStateMatching
	ConnStateGaming
)

type ConnContext struct {
	ID                int
	LoseHertbeatCount int
	Conn              goreactor.TCPConnection
	ConnState         ConnState

	// 下面的字段只有在ConnState为Gaming时有意义
	Gcontext *GameContext
}

type GameContext struct {
	RedConnID   int
	BlackConnID int
	Table       *commpackets.ChessTable
	TurnNow     commpackets.GameSide
}

// 包含所有连接的上下文, 用锁保护
var ConnMap map[int]*ConnContext

// 用来做自增连接id的计数器
var AtomicIDIncrease atomic.Int32

func init() {
	ConnMap = make(map[int]*ConnContext)
}

// 用来剔除过时的连接, 并发送心跳包
func OnTimeout() {
	fmt.Println("on timeout")
	var packet = commpackets.PacketHeartbeat{}
	heartPacketBytesWithHeader := tools.DoPackWith4BytesHeader(packet.MustMarshalToBytes())

	for k := range ConnMap {
		ConnMap[k].Conn.Send(heartPacketBytesWithHeader)
		ConnMap[k].LoseHertbeatCount++
		if ConnMap[k].LoseHertbeatCount >= commsettings.MaxLoseHeartbeat {
			ConnMap[k].Conn.ForceClose()
		}
	}
}

func OnConnect(c goreactor.TCPConnection) {
	fmt.Println("on connect")
	connID := int(AtomicIDIncrease.Add(1))
	connCtx := &ConnContext{ID: int(connID), LoseHertbeatCount: 0, Conn: c, ConnState: ConnStateNone, Gcontext: nil}

	ConnMap[connID] = connCtx

	c.SetContext("conn_id", connID)
	c.SetDisConnectedCallback(OnClose)
}

func OnClose(c goreactor.TCPConnection) {
	connID_, _ := c.GetContext("conn_id")
	connID := connID_.(int)
	if ConnMap[connID].ConnState == ConnStateGaming {
		var remoteID int
		if ConnMap[connID].Gcontext.BlackConnID == connID {
			remoteID = ConnMap[connID].Gcontext.RedConnID
		} else {
			remoteID = ConnMap[connID].Gcontext.BlackConnID
		}
		ConnMap[remoteID].Gcontext = nil
		ConnMap[remoteID].ConnState = ConnStateNone
		packet := commpackets.PacketServerRemoteLoseConnection{}
		packetBytesWithHeader := tools.DoPackWith4BytesHeader(packet.MustMarshalToBytes())
		ConnMap[remoteID].Conn.Send(packetBytesWithHeader)
	}
	delete(ConnMap, connID)
}

func OnMessage(c goreactor.TCPConnection, buf buffer.Buffer) {
	fmt.Println("on message")
	connID_, _ := c.GetContext("conn_id")
	connID := connID_.(int)
	packetBytes, ok := tools.IsBytesCompleteWith4BytesHeader(buf.Peek())
	if !ok {
		fmt.Println("not ok")
		return
	}

	packIface := commpackets.ServerParse(packetBytes)
	switch packet := packIface.(type) {
	// heartbeat包, 清空心跳包
	case *commpackets.PacketHeartbeat:
		ConnMap[connID].LoseHertbeatCount = 0
		return
	case *commpackets.PacketClientStartMatch:
		if ConnMap[connID].ConnState != ConnStateNone {
			// 协议错误
			fmt.Println("state none")
			c.ForceClose()
		}
		for _, v := range ConnMap {
			// 找到一个正在matching的连接, 做黑方
			if v.ID != connID && v.ConnState == ConnStateMatching {
				table := commpackets.NewDefaultChessTable()
				gameContext := GameContext{
					RedConnID:   connID,
					BlackConnID: v.ID,
					Table:       table,
					TurnNow:     commpackets.GameSideRed,
				}

				matchingPacket := commpackets.PacketServerMatching{}
				matchingPacketWithHeader := tools.DoPackWith4BytesHeader(matchingPacket.MustMarshalToBytes())

				packetForBlack := commpackets.PacketServerMatchedOK{Side: commpackets.GameSideBlack, Table: table}
				packetForBlackBytesWithHeader := tools.DoPackWith4BytesHeader(packetForBlack.MustMarshalToBytes())
				v.ConnState = ConnStateGaming
				v.Gcontext = &gameContext

				v.Conn.Send(matchingPacketWithHeader)
				v.Conn.Send(packetForBlackBytesWithHeader)

				packetForRed := commpackets.PacketServerMatchedOK{Side: commpackets.GameSideRed, Table: table}
				packetForRedBytesWithHeader := tools.DoPackWith4BytesHeader(packetForRed.MustMarshalToBytes())
				ConnMap[connID].ConnState = ConnStateGaming
				ConnMap[connID].Gcontext = &gameContext

				ConnMap[connID].Conn.Send(matchingPacketWithHeader)
				ConnMap[connID].Conn.Send(packetForRedBytesWithHeader)
				return
			}
		}

		// 找不到一个匹配的, 那么标记为正在匹配
		ConnMap[connID].ConnState = ConnStateMatching
		retPacket := commpackets.PacketServerMatching{}
		retPacketBytesWithHeader := tools.DoPackWith4BytesHeader(retPacket.MustMarshalToBytes())
		c.Send(retPacketBytesWithHeader)
		return
	case *commpackets.PacketClientMove:
		connCtx := ConnMap[connID]
		var connSide commpackets.GameSide
		var remoteSide commpackets.GameSide
		var remoteConnCtx *ConnContext
		if connCtx.Gcontext.BlackConnID == connID {
			connSide = commpackets.GameSideBlack
			remoteSide = commpackets.GameSideRed
			remoteConnCtx = ConnMap[connCtx.Gcontext.RedConnID]
		} else {
			connSide = commpackets.GameSideRed
			remoteSide = commpackets.GameSideBlack
			remoteConnCtx = ConnMap[connCtx.Gcontext.BlackConnID]
		}

		// 协议错误, 强行关闭连接
		if connCtx.ConnState != ConnStateGaming {
			c.ForceClose()
			return
		}

		if connCtx.Gcontext.TurnNow != connSide {
			retPacket := commpackets.PacketServerMoveResp{OK: false}
			msg := "it is not your turn"
			retPacket.ErrMsgOnFailed = &msg
			retPacketBytesWithHeader := tools.DoPackWith4BytesHeader(retPacket.MustMarshalToBytes())
			c.Send(retPacketBytesWithHeader)
			return
		}

		gameover, ok := tools.DoMove(connSide, connCtx.Gcontext.Table, packet.FromX, packet.FromY, packet.ToX, packet.ToY)
		if !ok {
			retPacket := commpackets.PacketServerMoveResp{OK: false}
			msg := "you cannot move like this"
			retPacket.ErrMsgOnFailed = &msg
			retPacketBytesWithHeader := tools.DoPackWith4BytesHeader(retPacket.MustMarshalToBytes())
			c.Send(retPacketBytesWithHeader)
			return
		}

		if !gameover {
			notifyPacket := commpackets.PacketServerNotifyRemoteMove{Table: connCtx.Gcontext.Table}
			notifyPacketBytesWithHeader := tools.DoPackWith4BytesHeader(notifyPacket.MustMarshalToBytes())
			remoteConnCtx.Conn.Send(notifyPacketBytesWithHeader)

			retPacket := commpackets.PacketServerMoveResp{OK: true, TableOnOK: connCtx.Gcontext.Table}
			retPacketBytesWithHeader := tools.DoPackWith4BytesHeader(retPacket.MustMarshalToBytes())
			connCtx.Gcontext.TurnNow = remoteSide
			c.Send(retPacketBytesWithHeader)
			return
		}

		// 游戏结束
		retPacket := commpackets.PacketServerGameOver{Table: connCtx.Gcontext.Table, WinnerSide: connSide}
		retPacketBytesWithHeader := tools.DoPackWith4BytesHeader(retPacket.MustMarshalToBytes())

		// 清空资源
		connCtx.ConnState = ConnStateNone
		remoteConnCtx.ConnState = ConnStateNone
		connCtx.Gcontext = nil
		remoteConnCtx.Gcontext = nil

		remoteConnCtx.Conn.Send(retPacketBytesWithHeader)
		return
	case nil:
		// 协议错误, 直接关闭
		c.ForceClose()
		return
	}
}
