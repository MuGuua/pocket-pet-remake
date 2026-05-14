package protocol

const HeaderSize = 26

type Packet struct {
	Length      uint32
	Cmd         uint16
	Seq         uint32
	TimestampMS uint64
	Code        uint32
	Checksum    uint32
	Body        []byte
}
