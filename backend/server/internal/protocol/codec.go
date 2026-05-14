package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
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

func EncodePacket(packet *Packet) ([]byte, error) {
	if packet == nil {
		return nil, fmt.Errorf("packet is nil")
	}

	packet.Checksum = checksum(packet.Cmd, packet.Seq, packet.TimestampMS, packet.Body)
	packet.Length = uint32(HeaderSize + len(packet.Body))

	buffer := bytes.NewBuffer(make([]byte, 0, packet.Length))
	fields := []any{
		packet.Length,
		packet.Cmd,
		packet.Seq,
		packet.TimestampMS,
		packet.Code,
		packet.Checksum,
	}
	for _, field := range fields {
		if err := binary.Write(buffer, binary.BigEndian, field); err != nil {
			return nil, err
		}
	}
	if len(packet.Body) > 0 {
		if _, err := buffer.Write(packet.Body); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func DecodePacket(data []byte) (*Packet, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("packet too small")
	}

	reader := bytes.NewReader(data)
	packet := &Packet{}
	fields := []any{
		&packet.Length,
		&packet.Cmd,
		&packet.Seq,
		&packet.TimestampMS,
		&packet.Code,
		&packet.Checksum,
	}
	for _, field := range fields {
		if err := binary.Read(reader, binary.BigEndian, field); err != nil {
			return nil, err
		}
	}

	if int(packet.Length) != len(data) {
		return nil, fmt.Errorf("packet length mismatch")
	}

	bodyLen := len(data) - HeaderSize
	if bodyLen > 0 {
		packet.Body = make([]byte, bodyLen)
		if _, err := io.ReadFull(reader, packet.Body); err != nil {
			return nil, err
		}
	}

	expectedChecksum := checksum(packet.Cmd, packet.Seq, packet.TimestampMS, packet.Body)
	if expectedChecksum != packet.Checksum {
		return nil, fmt.Errorf("invalid checksum")
	}

	return packet, nil
}

func checksum(cmd uint16, seq uint32, ts uint64, body []byte) uint32 {
	buffer := bytes.NewBuffer(make([]byte, 0, 14+len(body)))
	_ = binary.Write(buffer, binary.BigEndian, cmd)
	_ = binary.Write(buffer, binary.BigEndian, seq)
	_ = binary.Write(buffer, binary.BigEndian, ts)
	if len(body) > 0 {
		buffer.Write(body)
	}
	return crc32.ChecksumIEEE(buffer.Bytes())
}
