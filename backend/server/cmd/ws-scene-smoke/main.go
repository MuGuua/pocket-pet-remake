package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"

	"pocket-pet-remake/server/internal/protocol"
)

type loginResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		PlayerID   uint64 `json:"player_id"`
		PlayerName string `json:"player_name"`
		WSToken    string `json:"ws_token"`
	} `json:"data"`
}

func main() {
	httpBaseURL := flag.String("http", "http://127.0.0.1:8080", "HTTP base URL")
	wsURL := flag.String("ws", "ws://127.0.0.1:8080/ws", "WebSocket URL")
	account := flag.String("account", "demo", "login account")
	password := flag.String("password", "demo123", "login password")
	deviceID := flag.String("device-id", "smoke-client", "device identifier")
	flag.Parse()

	login, err := doLogin(*httpBaseURL, *account, *password, *deviceID)
	must(err)

	conn, _, err := websocket.DefaultDialer.Dial(*wsURL, nil)
	must(err)
	defer conn.Close()

	fmt.Printf("logged in as player=%d name=%s\n", login.Data.PlayerID, login.Data.PlayerName)
	must(writePacket(conn, protocol.CmdWSAuthReq, 1, protocol.WsAuthReq{
		WSToken:       login.Data.WSToken,
		ClientVersion: "ws-scene-smoke",
		DeviceID:      *deviceID,
	}))

	authPacket := mustReadPacket(conn)
	expectCmd(authPacket, protocol.CmdWSAuthResp)
	fmt.Printf("auth ok %s\n", string(authPacket.Body))

	must(writePacket(conn, protocol.CmdEnterWorldReq, 2, protocol.EnterWorldReq{}))
	enterPacket := mustReadPacket(conn)
	expectCmd(enterPacket, protocol.CmdEnterWorldResp)
	fmt.Printf("enter world %s\n", string(enterPacket.Body))

	must(transferScene(conn, 3, 1, 2))
	must(transferScene(conn, 4, 2, 3))
	fmt.Println("scene transfer smoke test passed: 1 -> 2 -> 3")
}

func doLogin(httpBaseURL, account, password, deviceID string) (*loginResponse, error) {
	body := strings.NewReader(fmt.Sprintf(`{"account":%q,"password":%q,"device_id":%q}`, account, password, deviceID))
	resp, err := http.Post(httpBaseURL+"/api/v1/auth/login", "application/json", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed loginResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	if parsed.Code != 200 || parsed.Data.WSToken == "" {
		return nil, fmt.Errorf("login failed: %s", string(payload))
	}
	return &parsed, nil
}

func transferScene(conn *websocket.Conn, seq uint32, fromSceneID, toSceneID uint32) error {
	if err := writePacket(conn, protocol.CmdMoveIntentReq, seq, protocol.MoveIntentReq{
		OpID:          seq,
		MoveSeq:       seq,
		SceneID:       fromSceneID,
		TargetSceneID: toSceneID,
	}); err != nil {
		return err
	}

	movePacket := mustReadPacket(conn)
	expectCmd(movePacket, protocol.CmdMoveIntentResp)
	var moveResp protocol.MoveIntentResp
	if err := protocol.UnmarshalBody(movePacket.Body, &moveResp); err != nil {
		return err
	}
	if !moveResp.Accepted || moveResp.SceneID != toSceneID {
		return fmt.Errorf("unexpected move response: %s", string(movePacket.Body))
	}
	fmt.Printf("move accepted %d -> %d %s\n", fromSceneID, toSceneID, string(movePacket.Body))

	resyncPacket := mustReadPacket(conn)
	expectCmd(resyncPacket, protocol.CmdWorldResyncPush)
	var resync protocol.WorldResyncPush
	if err := protocol.UnmarshalBody(resyncPacket.Body, &resync); err != nil {
		return err
	}
	if resync.SceneID != toSceneID {
		return fmt.Errorf("unexpected world resync: %s", string(resyncPacket.Body))
	}
	fmt.Printf("resync %d %s\n", toSceneID, string(resyncPacket.Body))
	return nil
}

func writePacket(conn *websocket.Conn, cmd uint16, seq uint32, payload any) error {
	packet, err := protocol.NewJSONPacket(cmd, seq, 0, payload)
	if err != nil {
		return err
	}
	encoded, err := protocol.EncodePacket(packet)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.BinaryMessage, encoded)
}

func mustReadPacket(conn *websocket.Conn) *protocol.Packet {
	messageType, raw, err := conn.ReadMessage()
	must(err)
	if messageType != websocket.BinaryMessage {
		fmt.Fprintf(os.Stderr, "unexpected websocket message type: %d\n", messageType)
		os.Exit(1)
	}
	packet, err := protocol.DecodePacket(raw)
	must(err)
	return packet
}

func expectCmd(packet *protocol.Packet, expected uint16) {
	if packet.Cmd != expected {
		fmt.Fprintf(os.Stderr, "unexpected cmd=%d expected=%d body=%s\n", packet.Cmd, expected, string(packet.Body))
		os.Exit(1)
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
