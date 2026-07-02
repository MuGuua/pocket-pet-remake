package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/teststub"
)

func TestRegisterHandler(t *testing.T) {
	handler := NewRegisterHandler(player.NewService(teststub.NewPlayerRepository(), nil, nil, nil))

	body, err := json.Marshal(map[string]string{
		"account":  "new_trainer",
		"password": "pwd123456",
		"gender":   "female",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d, body=%s", response.Code, http.StatusOK, response.Body.String())
	}

	var payload struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			PlayerID   uint64 `json:"player_id"`
			Account    string `json:"account"`
			PlayerName string `json:"player_name"`
			SkinID     string `json:"skin_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Msg != "success" {
		t.Fatalf("payload.Msg = %s, want success", payload.Msg)
	}
	if payload.Data.PlayerID == 0 {
		t.Fatal("payload.Data.PlayerID should not be zero")
	}
	if payload.Data.Account != "new_trainer" {
		t.Fatalf("payload.Data.Account = %s, want new_trainer", payload.Data.Account)
	}
	if payload.Data.PlayerName != "new_trainer" {
		t.Fatalf("payload.Data.PlayerName = %s, want new_trainer", payload.Data.PlayerName)
	}
	if payload.Data.SkinID != player.DefaultFemalePlayerSkinID {
		t.Fatalf("payload.Data.SkinID = %s, want %s", payload.Data.SkinID, player.DefaultFemalePlayerSkinID)
	}
}
