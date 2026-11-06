package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/world"
)

// 只实现本接口所需持久化，拒绝依赖其他玩法数据。
type adminAOIRepoStub struct {
	world.Repository
	value world.AOIConfig
}

func (r *adminAOIRepoStub) GetAOIConfig(context.Context) (world.AOIConfig, error) {
	return r.value, nil
}
func (r *adminAOIRepoStub) UpdateAOIConfig(_ context.Context, input world.UpdateAOIConfigInput) (world.AOIConfig, error) {
	r.value = world.AOIConfig{Enabled: input.Enabled, CellSizeMilli: input.CellSizeMilli, Radius: input.Radius, Reason: input.Reason, AdminUserID: input.AdminUserID}
	return r.value, nil
}

type adminSceneIndexStub struct{ world.MovementStateRepository }

func (*adminSceneIndexStub) MarkSceneOnline(context.Context, uint64, uint32, string, string, time.Duration) error {
	return nil
}
func (*adminSceneIndexStub) UnmarkSceneOnline(context.Context, uint64, string) error { return nil }
func (*adminSceneIndexStub) ListSceneOnlinePlayerIDs(context.Context, uint32) ([]uint64, error) {
	return []uint64{1, 2}, nil
}

// HTTP 权限、审计及配置/当前值分离在真实 handler 中核对。
func TestAdminAOIConfigAndSceneIndex(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	repo := &adminAOIRepoStub{value: world.AOIConfig{CellSizeMilli: 24000, Radius: 1}}
	service := world.NewService(repo)
	if err := service.RefreshAOIConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	service.SetMovementStateRepository(&adminSceneIndexStub{})
	handlers.WorldMovement.(*AdminWorldMovementHandler).worldService = service
	request := func(method, path, body string, authorized bool) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if authorized {
			req.Header.Set("Authorization", "Bearer "+issueAdminTokenForTest(t))
		}
		response := httptest.NewRecorder()
		handlers.WorldMovement.ServeHTTP(response, req)
		return response
	}
	if request(http.MethodGet, "/api/admin/world/aoi-config", "", false).Code == http.StatusOK {
		t.Fatal("unauthorized AOI read")
	}
	result := request(http.MethodPut, "/api/admin/world/aoi-config", `{"enabled":true,"cell_size_milli":12000,"radius":2,"reason":"测试"}`, true)
	if result.Code != 200 {
		t.Fatalf("AOI update failed %s", result.Body.String())
	}
	var envelope struct {
		Data map[string]world.AOIConfig `json:"data"`
	}
	if err := json.Unmarshal(result.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Data["configured"].Enabled || envelope.Data["active"].Enabled || repo.value.AdminUserID == 0 {
		t.Fatal("AOI current/configured/audit contract changed")
	}
	if request(http.MethodPut, "/api/admin/world/aoi-config", `{"cell_size_milli":1,"radius":0}`, true).Code != 400 {
		t.Fatal("invalid AOI update accepted")
	}
	online := request(http.MethodGet, "/api/admin/world/scene-online?scene_id=7", "", true)
	var counts struct {
		Data struct {
			Scene uint32 `json:"scene_id"`
			Count int    `json:"count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(online.Body.Bytes(), &counts); err != nil || online.Code != 200 || counts.Data.Scene != 7 || counts.Data.Count != 2 {
		t.Fatalf("shared index result %s %v", online.Body.String(), err)
	}
	if request(http.MethodGet, "/api/admin/world/scene-online?scene_id=invalid", "", true).Code != 400 {
		t.Fatal("invalid scene identifier accepted")
	}
}
