package commpackets

import "encoding/json"

// 提供给客户端的解析函数, 如果返回值为nil那么读的包为错误的包
// 此时服务端应当断开连接
func ClientParse(bs []byte) interface{} {
	header := PacketHeader{}
	err := json.Unmarshal(bs, &header)
	if err != nil {
		return nil
	}

	// 有可能发来的是四个字节表示0, 然后包本体一个字节也没有, 那么
	// 此时不对应任何一个包, 返回nil
	if header.Type == nil {
		return nil
	}

	switch *header.Type {
	case PacketTypeHeartbeat:
		return &PacketHeartbeat{}
	case PacketTypeServerGameOver:
		p := PacketServerGameOver{}
		json.Unmarshal(bs, &p)
		return &p
	case PacketTypeServerMatchedOK:
		p := PacketServerMatchedOK{}
		json.Unmarshal(bs, &p)
		return &p
	case PacketTypeServerMatching:
		p := PacketServerMatching{}
		json.Unmarshal(bs, &p)
		return &p
	case PacketTypeServerMoveResp:
		p := PacketServerMoveResp{}
		json.Unmarshal(bs, &p)
		return &p
	case PacketTypeServerRemoteLoseConnection:
		p := PacketServerRemoteLoseConnection{}
		json.Unmarshal(bs, &p)
		return &p
	case PacketTypeServerNotifyRemoteMove:
		p := PacketServerNotifyRemoteMove{}
		json.Unmarshal(bs, &p)
		return &p
	default:
		return nil
	}
}

// 提供给服务端的解析函数
func ServerParse(bs []byte) interface{} {
	header := PacketHeader{}
	err := json.Unmarshal(bs, &header)
	if err != nil {
		return nil
	}

	if header.Type == nil {
		return nil
	}

	switch *header.Type {
	case PacketTypeHeartbeat:
		return &PacketHeartbeat{}
	case PacketTypeClientMove:
		p := PacketClientMove{}
		json.Unmarshal(bs, &p)
		return &p
	case PacketTypeClientStartMatch:
		p := PacketClientStartMatch{}
		json.Unmarshal(bs, &p)
		return &p
	default:
		return nil
	}
}
