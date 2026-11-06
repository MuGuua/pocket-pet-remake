package protocol

import (
	"bytes"
	"fmt"
	"testing"
)

// 编码结果保留到包级变量，防止基准中未消费的返回值被编译器省略。
var benchmarkPacketBytes []byte
var benchmarkDecodedPacket *Packet

// 覆盖心跳、移动包和较大面板载荷；仅测二进制封包，不把 JSON 序列化混入结果。
func BenchmarkPacketCodec(b *testing.B) {
	for _, size := range []int{0, 256, 65536} {
		packet := NewPacket(CmdEntityMovePush, 17, 0, bytes.Repeat([]byte{'x'}, size))
		encoded, err := EncodePacket(packet)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("encode_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(encoded)))
			for n := 0; n < b.N; n++ {
				var err error
				benchmarkPacketBytes, err = EncodePacket(packet)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("decode_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(encoded)))
			for n := 0; n < b.N; n++ {
				var err error
				benchmarkDecodedPacket, err = DecodePacket(encoded)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
