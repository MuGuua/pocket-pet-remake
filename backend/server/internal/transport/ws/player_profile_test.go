package wstransport

import (
	"context"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/protocol"
	"pocket-pet-remake/server/internal/teststub"
	"testing"
)

// 展示能力仓储为面板返回新数值，并分别记录只读与旧档案读取入口。
type displayPlayerRepository struct {
	*teststub.PlayerRepository
	displayReads int
	fullReads    int
}

func (r *displayPlayerRepository) FindDisplayProfile(ctx context.Context, id uint64) (*player.Profile, error) {
	r.displayReads++
	p, err := r.PlayerRepository.FindByPlayerID(ctx, id)
	if err == nil && p != nil {
		p.ATK = 987
	}
	return p, err
}
func (r *displayPlayerRepository) FindByPlayerID(ctx context.Context, id uint64) (*player.Profile, error) {
	r.fullReads++
	return r.PlayerRepository.FindByPlayerID(ctx, id)
}

// 人物面板只读入口接入真实路由，回包命令、序号和身份字段保持原协议。
func TestPlayerProfileUsesDisplayRepository(t *testing.T) {
	repo := &displayPlayerRepository{PlayerRepository: teststub.NewPlayerRepository()}
	svc := player.NewService(repo, nil, nil, nil)
	id, router, _, conn := buildWorldRouterWithPlayerServiceForTest(t, svc)
	repo.displayReads = 0
	repo.fullReads = 0
	mustHandleJSONPacket(t, router, conn, protocol.CmdPlayerProfileReq, 910, protocol.PlayerProfileReq{})
	if len(conn.packets) != 1 || conn.packets[0].Cmd != protocol.CmdPlayerProfileResp || conn.packets[0].Seq != 910 {
		t.Fatal("profile response contract changed")
	}
	var result protocol.PlayerProfileResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &result); err != nil {
		t.Fatal(err)
	}
	if result.Player.PlayerID != id || result.Player.ATK != 987 || repo.displayReads != 1 || repo.fullReads != 0 {
		t.Fatal("profile request did not use read-only display route")
	}
}
