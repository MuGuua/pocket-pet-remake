package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"time"
)

func NewPacket(cmd uint16, seq uint32, code uint32, body []byte) *Packet {
	return &Packet{
		Cmd:         cmd,
		Seq:         seq,
		TimestampMS: uint64(time.Now().UnixMilli()),
		Code:        code,
		Body:        body,
	}
}

func NewJSONPacket(cmd uint16, seq uint32, code uint32, payload any) (*Packet, error) {
	body, err := MarshalBody(payload)
	if err != nil {
		return nil, err
	}
	return NewPacket(cmd, seq, code, body), nil
}

func MarshalBody(payload any) ([]byte, error) {
	if payload == nil {
		return nil, nil
	}
	return json.Marshal(payload)
}

func UnmarshalBody(body []byte, target any) error {
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, target)
}

// 按固定 26 字节包头的大端布局直接封包，返回独立字节，仍回填包的长度和校验字段。
// 字段偏移依次为长度 0、命令 4、序号 6、时间戳 10、业务码 18、校验和 22、载荷 26。
func EncodePacket(packet *Packet) ([]byte, error) {
	if packet == nil {
		return nil, fmt.Errorf("packet is nil")
	}
	encoded := make([]byte, HeaderSize+len(packet.Body))
	packet.Length = uint32(len(encoded))
	binary.BigEndian.PutUint32(encoded[0:4], packet.Length)
	binary.BigEndian.PutUint16(encoded[4:6], packet.Cmd)
	binary.BigEndian.PutUint32(encoded[6:10], packet.Seq)
	binary.BigEndian.PutUint64(encoded[10:18], packet.TimestampMS)
	binary.BigEndian.PutUint32(encoded[18:22], packet.Code)
	copy(encoded[HeaderSize:], packet.Body)
	packet.Checksum = checksum(encoded[4:18], encoded[HeaderSize:])
	binary.BigEndian.PutUint32(encoded[22:26], packet.Checksum)
	return encoded, nil
}

// 先检查长度与校验，再复制载荷；成功结果仍独立于输入缓冲，不能因后续复用输入而改变。
// 不扩大校验覆盖范围，继续只校验命令、序号、时间戳与载荷，保持旧客户端协议兼容。
func DecodePacket(data []byte) (*Packet, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("packet too small")
	}
	length := binary.BigEndian.Uint32(data[0:4])
	if uint64(length) != uint64(len(data)) {
		return nil, fmt.Errorf("packet length mismatch")
	}
	receivedChecksum := binary.BigEndian.Uint32(data[22:26])
	if checksum(data[4:18], data[HeaderSize:]) != receivedChecksum {
		return nil, fmt.Errorf("invalid checksum")
	}
	packet := &Packet{
		Length:      length,
		Cmd:         binary.BigEndian.Uint16(data[4:6]),
		Seq:         binary.BigEndian.Uint32(data[6:10]),
		TimestampMS: binary.BigEndian.Uint64(data[10:18]),
		Code:        binary.BigEndian.Uint32(data[18:22]),
		Checksum:    receivedChecksum,
	}
	if len(data) > HeaderSize {
		packet.Body = make([]byte, len(data)-HeaderSize)
		copy(packet.Body, data[HeaderSize:])
	}
	return packet, nil
}

// 分段计算同一条 CRC32/IEEE 字节流，不再为了校验而拼接、复制整份载荷。
// header 必须是大端编码的命令、序号、时间戳共 14 字节；body 可以为空。
func checksum(header []byte, body []byte) uint32 {
	return crc32.Update(crc32.ChecksumIEEE(header), crc32.IEEETable, body)
}
