package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"pocket-pet-remake/server/internal/config"
	redisrepo "pocket-pet-remake/server/internal/data/redis"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/protocol"
	wstransport "pocket-pet-remake/server/internal/transport/ws"
)

// 这里只替代静态摘要和数据库配置，移动、连接、Redis Stream 与租约均使用正式实现。
type relayIntegrationWorldRepo struct {
	world.Repository
	config world.AOIConfig
}

func (r *relayIntegrationWorldRepo) GetAOIConfig(context.Context) (world.AOIConfig, error) {
	return r.config, nil
}
func (*relayIntegrationWorldRepo) UpdateAOIConfig(context.Context, world.UpdateAOIConfigInput) (world.AOIConfig, error) {
	return world.AOIConfig{}, nil
}

type relayIntegrationPlayerRepo struct{ player.Repository }

func (*relayIntegrationPlayerRepo) ListWorldSummariesByPlayerIDs(_ context.Context, ids []uint64) ([]player.WorldSummary, error) {
	var values []player.WorldSummary
	for _, id := range ids {
		values = append(values, player.WorldSummary{PlayerID: id, SceneID: 7, Name: fmt.Sprint("player", id)})
	}
	return values, nil
}

// 两台真实 WS 服务与独立会话/Redis 客户端，在同一 Redis 命名空间交换世界事件。
func TestWorldRelayRealWebSocketRedis(t *testing.T) {
	if os.Getenv("POCKET_REDIS_TEST_ADDR") == "" || os.Getenv("POCKET_BAG_PAGE_TEST_DSN") == "" {
		t.Skip("requires isolated local PostgreSQL/Redis fixtures")
	}
	for _, aoi := range []bool{false, true} {
		t.Run(fmt.Sprint(aoi), func(t *testing.T) {
			db, _ := movementPipelineFixture(t)
			var prefix string
			if err := db.QueryRow(`SELECT current_schema()`).Scan(&prefix); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var relays, handlers sync.WaitGroup
			clients := make([]*websocket.Conn, 0, 2)
			connections := make([]*wstransport.Connection, 0, 2)
			servers := make([]*httptest.Server, 0, 2)
			redisClients := make([]*redisrepo.RuntimeClient, 0, 2)
			t.Cleanup(func() {
				cancel()
				for _, client := range clients {
					client.Close()
				}
				for _, conn := range connections {
					conn.Close()
				}
				for _, server := range servers {
					server.Close()
				}
				handlers.Wait()
				relays.Wait()
				for _, client := range redisClients {
					client.Close()
				}
			})
			logger := log.New(io.Discard, "", 0)
			for index := 0; index < 2; index++ {
				id := uint64(index + 1)
				client, err := redisrepo.Open(config.RedisConfig{Addr: os.Getenv("POCKET_REDIS_TEST_ADDR")})
				if err != nil {
					t.Fatal(err)
				}
				redisClients = append(redisClients, client)
				stateRepo := redisrepo.NewMovementStateRepository(client, prefix)
				service := world.NewService(&relayIntegrationWorldRepo{config: world.AOIConfig{Enabled: aoi, RelayEnabled: true, CellSizeMilli: 1000, Radius: 1}})
				if err := service.RefreshAOIConfig(ctx); err != nil {
					t.Fatal(err)
				}
				service.SetMovementStateRepository(stateRepo)
				sessions := session.NewService(logger, 100*time.Millisecond, 10*time.Second)
				handler := wstransport.NewWorldHandler(sessions, player.NewService(&relayIntegrationPlayerRepo{}, nil, nil, nil), nil, nil, nil, service, nil, nil)
				if err := handler.ConfigureWorldRelay(redisrepo.NewSceneEventBus(client, prefix)); err != nil {
					t.Fatal(err)
				}
				sessions.AddDisconnectHandler(handler.HandleOfflineSession)
				router := wstransport.NewRouter(nil, handler, nil, nil, nil, nil, nil, nil, sessions)
				relays.Add(1)
				go func() { defer relays.Done(); handler.RunWorldRelay(ctx) }()
				accepted := make(chan *wstransport.Connection, 1)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handlers.Add(1)
					defer handlers.Done()
					socket, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
					if err != nil {
						return
					}
					conn := wstransport.NewConnection(fmt.Sprint("relay-node", id), socket)
					defer func() { sessions.Disconnect(conn.ID()); conn.Close() }()
					accepted <- conn
					for {
						kind, raw, err := socket.ReadMessage()
						if err != nil {
							return
						}
						if kind == websocket.BinaryMessage {
							if err := router.Handle(conn, raw); err != nil {
								return
							}
						}
					}
				}))
				servers = append(servers, server)
				socket, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
				if err != nil {
					t.Fatal(err)
				}
				clients = append(clients, socket)
				conn := <-accepted
				connections = append(connections, conn)
				bound, err := sessions.Bind(id, conn)
				if err != nil {
					t.Fatal(err)
				}
				if err := service.InitializeMovementState(ctx, world.MovementState{PlayerID: id, SessionID: bound.ID, SceneID: 7, SceneVersion: 1, PositionVersion: 1, Speed: 1000}); err != nil {
					t.Fatal(err)
				}
				relaySendMove(t, socket, 1, 0, true)
			}
			relayAwaitCommand(t, clients[0], protocol.CmdEntityEnterPush)
			relayAwaitCommand(t, clients[1], protocol.CmdEntityEnterPush)
			relaySendMove(t, clients[0], 2, 500, true)
			moved := relayAwaitCommand(t, clients[1], protocol.CmdEntityMovePush)
			var state protocol.EntityMovePush
			if err := protocol.UnmarshalBody(moved.Body, &state); err != nil || state.EntityID != 1 {
				t.Fatal("cross-instance movement failed", err)
			}
			// 可能先有暖身帧，继续等本次实际序号。
			for state.MoveSeq < 2 {
				moved = relayAwaitCommand(t, clients[1], protocol.CmdEntityMovePush)
				if err := protocol.UnmarshalBody(moved.Body, &state); err != nil {
					t.Fatal(err)
				}
			}
			if state.MoveSeq != 2 || state.PrecisePos.X != 500 {
				t.Fatal("movement payload changed")
			}
			relaySendMove(t, clients[0], 3, 10000, true)
			if aoi {
				relayAwaitCommand(t, clients[1], protocol.CmdEntityLeavePush)
				relaySendMove(t, clients[0], 4, 0, false)
				relayAwaitCommand(t, clients[1], protocol.CmdEntityEnterPush)
			} else {
				relaySendMove(t, clients[0], 4, 0, false)
			}
			for {
				packet := relayAwaitCommand(t, clients[1], protocol.CmdEntityMovePush)
				if err := protocol.UnmarshalBody(packet.Body, &state); err != nil {
					t.Fatal(err)
				}
				if state.MoveSeq == 4 {
					if state.Moving {
						t.Fatal("stop lost")
					}
					break
				}
			}
			clients[0].Close()
			relayAwaitCommand(t, clients[1], protocol.CmdEntityLeavePush)
		})
	}
}

// 测试端发送仍走正式协议编码与真实 socket，不直接调用广播绕过移动状态确认。
func relaySendMove(t *testing.T, socket *websocket.Conn, seq uint32, x int32, moving bool) {
	t.Helper()
	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, seq, 0, protocol.MoveIntentReq{MoveSeq: seq, SceneID: 7, TargetPos: &protocol.Vec2i{X: x / 1000}, PrecisePos: &protocol.Vec2i{X: x}, Moving: &moving, Facing: &protocol.Vec2i{X: 1}, ClientTick: int64(seq) * 100})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatal(err)
	}
	if err := socket.WriteMessage(websocket.BinaryMessage, raw); err != nil {
		t.Fatal(err)
	}
}

// 实际观察端等待目标通知，任何异常关闭/解析失败都不能被当作测试成功。
func relayAwaitCommand(t *testing.T, socket *websocket.Conn, command uint16) *protocol.Packet {
	t.Helper()
	socket.SetReadDeadline(time.Now().Add(4 * time.Second))
	for {
		_, raw, err := socket.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		packet, err := protocol.DecodePacket(raw)
		if err != nil {
			t.Fatal(err)
		}
		if packet.Cmd == command {
			return packet
		}
	}
}
