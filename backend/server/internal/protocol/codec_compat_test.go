package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math/rand"
	"reflect"
	"testing"
)

// 固定随机种子覆盖任意二进制载荷、整数边界和大包，与改动前编码逐字节比较。
func TestPacketCodecLegacyCompatibility(t *testing.T) {
	random := rand.New(rand.NewSource(42))
	for index := 0; index < 256; index++ {
		size := index
		if index == 255 {
			size = 65536
		}
		body := make([]byte, size)
		_, _ = random.Read(body)
		original := Packet{Cmd: uint16(random.Uint32()), Seq: random.Uint32(), TimestampMS: random.Uint64(), Code: random.Uint32(), Body: body}
		if index == 0 {
			original = Packet{}
		}
		if index == 1 {
			original.Cmd = ^uint16(0)
			original.Seq = ^uint32(0)
			original.TimestampMS = ^uint64(0)
			original.Code = ^uint32(0)
		}
		legacy := original
		expected, err := legacyEncodePacket(&legacy)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodePacket(&original)
		if err != nil || !bytes.Equal(encoded, expected) || original.Length != legacy.Length || original.Checksum != legacy.Checksum {
			t.Fatalf("case %d wire/metadata mismatch: %v", index, err)
		}
		assertLegacyDecode(t, encoded)
	}
	if _, err := EncodePacket(nil); err == nil || err.Error() != "packet is nil" {
		t.Fatal("nil encode behavior changed")
	}
}

// 输入/输出仍各自拥有载荷，不用零拷贝破坏队列共享与后续复用缓冲的边界。
func TestPacketCodecBufferOwnership(t *testing.T) {
	packet := NewPacket(2013, 9, 0, []byte("original"))
	encoded, err := EncodePacket(packet)
	if err != nil {
		t.Fatal(err)
	}
	packet.Body[0] = 'x'
	decoded, err := DecodePacket(encoded)
	if err != nil || string(decoded.Body) != "original" {
		t.Fatalf("encoded bytes alias original body: %v", err)
	}
	encoded[HeaderSize] = 'y'
	if string(decoded.Body) != "original" {
		t.Fatal("decoded body aliases receive buffer")
	}
	empty, _ := EncodePacket(NewPacket(2013, 0, 0, nil))
	decoded, err = DecodePacket(empty)
	if err != nil || decoded.Body != nil {
		t.Fatal("empty body nil behavior changed")
	}
}

// 长度、截断和覆盖范围内的篡改仍返回相同错误，业务码继续沿用旧协议的非校验字段语义。
func TestPacketCodecMalformedCompatibility(t *testing.T) {
	valid, _ := legacyEncodePacket(NewPacket(2013, 7, 0, []byte("payload")))
	for size := 0; size < len(valid); size++ {
		assertLegacyDecode(t, valid[:size])
	}
	for index := range valid {
		changed := bytes.Clone(valid)
		changed[index] ^= 0x80
		assertLegacyDecode(t, changed)
	}
	assertLegacyDecode(t, append(bytes.Clone(valid), 0))
}

// 模糊输入同时经过旧解码器和新解码器，比较错误及完整输出，防止边界包引发协议偏差或 panic。
func FuzzPacketDecodeLegacyCompatibility(f *testing.F) {
	valid, _ := legacyEncodePacket(NewPacket(2013, 7, 0, []byte(`{"moving":false}`)))
	f.Add(valid)
	f.Add([]byte{})
	f.Add(make([]byte, HeaderSize))
	f.Fuzz(func(t *testing.T, data []byte) { assertLegacyDecode(t, data) })
}

// 冻结解码器作为独立兼容性参照，比较错误文本以覆盖原连接层可观察的失败行为。
func assertLegacyDecode(t *testing.T, data []byte) {
	t.Helper()
	expected, oldErr := legacyDecodePacket(data)
	decoded, err := DecodePacket(data)
	if (oldErr == nil) != (err == nil) {
		t.Fatalf("decode error changed: old=%v new=%v", oldErr, err)
	}
	if oldErr != nil {
		if oldErr.Error() != err.Error() {
			t.Fatalf("decode error text changed: old=%v new=%v", oldErr, err)
		}
	} else if !reflect.DeepEqual(decoded, expected) {
		t.Fatal("decoded packet changed")
	}
}

// 以下保留改动前的流式实现，仅作为测试参照；正式编解码没有双路径或额外兼容分支。
func legacyEncodePacket(packet *Packet) ([]byte, error) {
	if packet == nil {
		return nil, fmt.Errorf("packet is nil")
	}

	packet.Checksum = legacyChecksum(packet.Cmd, packet.Seq, packet.TimestampMS, packet.Body)
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

func legacyDecodePacket(data []byte) (*Packet, error) {
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

	expectedChecksum := legacyChecksum(packet.Cmd, packet.Seq, packet.TimestampMS, packet.Body)
	if expectedChecksum != packet.Checksum {
		return nil, fmt.Errorf("invalid checksum")
	}

	return packet, nil
}

func legacyChecksum(cmd uint16, seq uint32, ts uint64, body []byte) uint32 {
	buffer := bytes.NewBuffer(make([]byte, 0, 14+len(body)))
	_ = binary.Write(buffer, binary.BigEndian, cmd)
	_ = binary.Write(buffer, binary.BigEndian, seq)
	_ = binary.Write(buffer, binary.BigEndian, ts)
	if len(body) > 0 {
		buffer.Write(body)
	}
	return crc32.ChecksumIEEE(buffer.Bytes())
}
