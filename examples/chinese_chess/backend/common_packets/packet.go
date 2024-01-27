package commpackets

import "encoding/json"

type PacketType int

const (
	// 心跳包
	PacketTypeHeartbeat PacketType = iota
	// 客户端要求开始匹配
	PacketTypeClientStartMatch
	// 服务端表示已经开始匹配
	PacketTypeServerMatching
	// 匹配完毕, 即将开始游戏
	PacketTypeServerMatchedOK
	// 客户端发送下棋的消息
	PacketTypeClientMove
	// 服务端告知用户下棋结果
	PacketTypeServerMoveResp
	// 通知游戏结束
	PacketTypeServerGameOver
	// 通知对方掉线
	PacketTypeServerRemoteLoseConnection
	// 对方下棋下好了
	PacketTypeServerNotifyRemoteMove
)

type PacketHeader struct {
	Type *PacketType `json:"type"`
}

type PacketHeartbeat struct {
	PacketHeader
}

func (p *PacketHeartbeat) MustMarshalToBytes() []byte {
	i := PacketTypeHeartbeat
	p.Type = &i
	bs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return bs
}

type PacketClientStartMatch struct {
	PacketHeader
}

func (p *PacketClientStartMatch) MustMarshalToBytes() []byte {
	i := PacketTypeClientStartMatch
	p.Type = &i
	bs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return bs
}

type PacketServerMatching struct {
	PacketHeader
}

func (p *PacketServerMatching) MustMarshalToBytes() []byte {
	i := PacketTypeServerMatching
	p.Type = &i
	bs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return bs
}

type PacketServerMatchedOK struct {
	PacketHeader
	Side  GameSide    `json:"game_side"`
	Table *ChessTable `json:"game_table"`
}

func (p *PacketServerMatchedOK) MustMarshalToBytes() []byte {
	i := PacketTypeServerMatchedOK
	p.Type = &i
	bs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return bs
}

type PacketClientMove struct {
	PacketHeader
	FromX int `json:"from_x"`
	FromY int `json:"from_y"`
	ToX   int `json:"to_x"`
	ToY   int `json:"to_y"`
}

func (p *PacketClientMove) MustMarshalToBytes() []byte {
	i := PacketTypeClientMove
	p.Type = &i
	bs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return bs
}

type PacketServerMoveResp struct {
	PacketHeader
	OK bool `json:"ok"`
	// 下面的字段只有在OK == true的时候出现
	TableOnOK *ChessTable `json:"table,omitempty"`

	// 下面的字段只有在OK == false的时候出现
	ErrMsgOnFailed *string `json:"errmsg,omitempty"`
}

func (p *PacketServerMoveResp) MustMarshalToBytes() []byte {
	i := PacketTypeServerMoveResp
	p.Type = &i
	bs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return bs
}

type PacketServerGameOver struct {
	PacketHeader
	Table      *ChessTable `json:"final_table"`
	WinnerSide GameSide    `json:"winner_side"`
}

func (p *PacketServerGameOver) MustMarshalToBytes() []byte {
	i := PacketTypeServerGameOver
	p.Type = &i
	bs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return bs
}

type PacketServerRemoteLoseConnection struct {
	PacketHeader
}

func (p *PacketServerRemoteLoseConnection) MustMarshalToBytes() []byte {
	i := PacketTypeServerRemoteLoseConnection
	p.Type = &i
	bs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return bs
}

type PacketServerNotifyRemoteMove struct {
	PacketHeader
	Table *ChessTable
}

func (p *PacketServerNotifyRemoteMove) MustMarshalToBytes() []byte {
	i := PacketTypeServerNotifyRemoteMove
	p.Type = &i
	bs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return bs
}
