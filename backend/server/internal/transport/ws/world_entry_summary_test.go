package wstransport

import (
	"context"
	"errors"
	"io"
	"log"
	"reflect"
	"strconv"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/protocol"
)

// 仅提供世界摘要能力；嵌入的完整仓储不得被该广播路径调用。
type entrySummaryPlayerRepo struct {
	player.Repository
	requests [][]uint64
	summary  player.WorldSummary
	err      error
}

// 记录真实领域服务收到的读取范围，避免只检查最终包而遗漏全场景重复查询。
func (r *entrySummaryPlayerRepo) ListWorldSummariesByPlayerIDs(_ context.Context, ids []uint64) ([]player.WorldSummary, error) {
	r.requests = append(r.requests, append([]uint64(nil), ids...))
	if r.err != nil {
		return nil, r.err
	}
	var result []player.WorldSummary
	for _, id := range ids {
		if id == r.summary.PlayerID && id != 0 {
			result = append(result, r.summary)
		}
	}
	return result, nil
}

// 跟随宠物同样必须只读取进入者，不借用完整编队或战斗快照。
type entrySummaryPetRepo struct {
	pet.Repository
	requests [][]uint64
	value    pet.WorldFollowingPet
	err      error
}

// 返回与数据库轻量仓储相同的首只跟随宠物结果，保存请求副本用于范围验证。
func (r *entrySummaryPetRepo) ListWorldFollowingPetsByPlayerIDs(_ context.Context, ids []uint64) (map[uint64]pet.WorldFollowingPet, error) {
	r.requests = append(r.requests, append([]uint64(nil), ids...))
	if r.err != nil {
		return nil, r.err
	}
	result := make(map[uint64]pet.WorldFollowingPet)
	for _, id := range ids {
		if id == r.value.PlayerID && id != 0 {
			result[id] = r.value
		}
	}
	return result, nil
}

// 无论场景里有多少人，新增玩家广播只查本人和其宠物；接收范围和所有展示字段保持完整。
func TestWorldEntryReadsOnlyEnteringPlayer(t *testing.T) {
	for _, observers := range []int{20, 50, 100} {
		t.Run(strconv.Itoa(observers), func(t *testing.T) {
			players := &entrySummaryPlayerRepo{summary: player.WorldSummary{
				PlayerID: 1001, Name: "进入者", Level: 8, Exp: 123, SceneID: 7, PosX: 4, PosY: 9,
				HP: 20, HPMax: 40, Vigor: 5, VigorMax: 10, Spirit: 7, SpiritMax: 14, SkinID: "player_skin",
			}}
			pets := &entrySummaryPetRepo{value: pet.WorldFollowingPet{
				PlayerID: 1001, PetUID: 2001, PetID: 101, Name: "跟随宠物", SkinID: "pet_skin", Level: 6,
				Exp: 55, HP: 12, HPMax: 30, Spirit: 8, SpiritMax: 16,
			}}
			sessions := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
			handler := NewWorldHandler(sessions, player.NewService(players, nil, nil, nil), pet.NewService(pets, nil, nil, nil), nil, nil, nil, nil, nil)
			peers := make([]*fakeConn, observers)
			for index := range peers {
				id := uint64(index + 1)
				peers[index] = &fakeConn{id: "observer-" + strconv.Itoa(index)}
				if _, err := sessions.Bind(id, peers[index]); err != nil {
					t.Fatal(err)
				}
				handler.setPlayerScenePresence(id, 7)
			}
			self := &fakeConn{id: "entry-self"}
			other := &fakeConn{id: "entry-other-map"}
			if _, err := sessions.Bind(1001, self); err != nil {
				t.Fatal(err)
			}
			if _, err := sessions.Bind(9001, other); err != nil {
				t.Fatal(err)
			}
			handler.setPlayerScenePresence(9001, 8)
			handler.enterPlayerScene(context.Background(), 1001, 7)
			wantIDs := [][]uint64{{1001}}
			if !reflect.DeepEqual(players.requests, wantIDs) || !reflect.DeepEqual(pets.requests, wantIDs) {
				t.Fatalf("entry read scope: players=%v pets=%v", players.requests, pets.requests)
			}
			want := protocol.EntityBrief{
				EntityID: 1001, PlayerID: 1001, EntityType: 1, Pos: protocol.Vec2i{X: 4, Y: 9},
				Name: "进入者", Level: 8, Exp: 123, HP: 20, HPMax: 40, Vigor: 5, VigorMax: 10,
				Spirit: 7, SpiritMax: 14, SkinID: "player_skin",
				FollowingPet: &protocol.PetBrief{PetUID: 2001, PetID: 101, Name: "跟随宠物", SkinID: "pet_skin", Level: 6, Exp: 55, HP: 12, HPMax: 30, Spirit: 8, SpiritMax: 16},
			}
			for _, peer := range peers {
				if len(peer.packets) != 1 || peer.packets[0].Cmd != protocol.CmdEntityEnterPush {
					t.Fatal("entry notification missing")
				}
				var push protocol.EntityEnterPush
				if err := protocol.UnmarshalBody(peer.packets[0].Body, &push); err != nil {
					t.Fatal(err)
				}
				if push.SceneID != 7 || !reflect.DeepEqual(push.Entity, want) {
					t.Fatalf("entry payload changed: %+v", push)
				}
			}
			if len(self.packets) != 0 || len(other.packets) != 0 {
				t.Fatal("self/other-scene broadcast boundary changed")
			}
			// 同场景再次登记不重复读取或推送；断开后重新进入则获取最新源摘要。
			handler.enterPlayerScene(context.Background(), 1001, 7)
			if len(players.requests) != 1 || len(pets.requests) != 1 {
				t.Fatal("duplicate entry unexpectedly queried summaries")
			}
			handler.HandleSessionDisconnect(1001)
			players.summary.Name = "更新后的进入者"
			pets.value.SkinID = "updated_pet_skin"
			handler.enterPlayerScene(context.Background(), 1001, 7)
			var latest protocol.EntityEnterPush
			if err := protocol.UnmarshalBody(peers[0].packets[2].Body, &latest); err != nil {
				t.Fatal(err)
			}
			if latest.Entity.Name != "更新后的进入者" || latest.Entity.FollowingPet == nil || latest.Entity.FollowingPet.SkinID != "updated_pet_skin" {
				t.Fatal("reentry reused stale summary")
			}
		})
	}
}

// 缺失/失败不广播空实体；宠物失败仍按原行为发送无跟随宠物的玩家摘要。
func TestWorldEntrySummaryReadFailures(t *testing.T) {
	for _, mode := range []string{"missing", "player_error", "pet_error", "no_pet_service"} {
		t.Run(mode, func(t *testing.T) {
			players := &entrySummaryPlayerRepo{summary: player.WorldSummary{PlayerID: 1001, Name: "进入者"}}
			pets := &entrySummaryPetRepo{err: errors.New("pet read failed")}
			if mode == "missing" {
				players.summary.PlayerID = 0
			}
			if mode == "player_error" {
				players.err = errors.New("player read failed")
			}
			sessions := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
			handler := NewWorldHandler(sessions, player.NewService(players, nil, nil, nil), pet.NewService(pets, nil, nil, nil), nil, nil, nil, nil, nil)
			if mode == "no_pet_service" {
				handler.petService = nil
			}
			peer := &fakeConn{id: "entry-error-peer"}
			if _, err := sessions.Bind(2, peer); err != nil {
				t.Fatal(err)
			}
			handler.setPlayerScenePresence(2, 7)
			handler.enterPlayerScene(context.Background(), 1001, 7)
			if mode == "missing" || mode == "player_error" {
				if len(peer.packets) != 0 {
					t.Fatal("missing/failed entrant was broadcast")
				}
				if mode == "player_error" && len(pets.requests) != 0 {
					t.Fatal("pet query continued after player error")
				}
				return
			}
			if len(peer.packets) != 1 {
				t.Fatal("pet failure suppressed player entry")
			}
			var push protocol.EntityEnterPush
			if err := protocol.UnmarshalBody(peer.packets[0].Body, &push); err != nil {
				t.Fatal(err)
			}
			if push.Entity.PlayerID != 1001 || push.Entity.FollowingPet != nil {
				t.Fatal("pet failure behavior changed")
			}
		})
	}
}
