package tools

import (
	"encoding/binary"
	"io"
	"net"
)

func ReadPacketBytesWith4BytesHeader(conn net.Conn) ([]byte, error) {
	lengthBytes := make([]byte, 4)
	_, err := io.ReadFull(conn, lengthBytes)
	if err != nil {
		return nil, err
	}
	packetBytes := make([]byte, binary.BigEndian.Uint32(lengthBytes))

	_, err = io.ReadFull(conn, packetBytes)
	if err != nil {
		return nil, err
	}

	return packetBytes, nil
}

// 做封包操作
func DoPackWith4BytesHeader(bs []byte) []byte {
	buf := make([]byte, 4, 4+len(bs))
	binary.BigEndian.PutUint32(buf, uint32(len(bs)))
	buf = append(buf, bs...)
	return buf
}
