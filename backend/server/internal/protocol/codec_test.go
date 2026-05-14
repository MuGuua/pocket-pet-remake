package protocol

import "testing"

func TestPacketEncodeDecodeRoundTrip(t *testing.T) {
	packet, err := NewJSONPacket(CmdHeartbeatResp, 7, 0, HeartbeatResp{ServerTimeMS: 123456789})
	if err != nil {
		t.Fatalf("NewJSONPacket() error = %v", err)
	}

	encoded, err := EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	decoded, err := DecodePacket(encoded)
	if err != nil {
		t.Fatalf("DecodePacket() error = %v", err)
	}

	if decoded.Cmd != CmdHeartbeatResp {
		t.Fatalf("decoded.Cmd = %d, want %d", decoded.Cmd, CmdHeartbeatResp)
	}
	if decoded.Seq != 7 {
		t.Fatalf("decoded.Seq = %d, want 7", decoded.Seq)
	}

	var payload HeartbeatResp
	if err := UnmarshalBody(decoded.Body, &payload); err != nil {
		t.Fatalf("UnmarshalBody() error = %v", err)
	}
	if payload.ServerTimeMS != 123456789 {
		t.Fatalf("payload.ServerTimeMS = %d, want 123456789", payload.ServerTimeMS)
	}
}
