package httptransport

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/platform/errcode"
)

type RegisterHandler struct {
	playerService *player.Service
}

type registerRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Gender   string `json:"gender"`
}

type registerResponse struct {
	PlayerID   uint64 `json:"player_id"`
	Account    string `json:"account"`
	PlayerName string `json:"player_name"`
	SkinID     string `json:"skin_id"`
}

func NewRegisterHandler(playerService *player.Service) *RegisterHandler {
	return &RegisterHandler{playerService: playerService}
}

func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errcode.HTTPInvalidRequest, "method not allowed", nil)
		return
	}
	if h.playerService == nil {
		writeJSON(w, http.StatusInternalServerError, errcode.HTTPInternalServer, "register service unavailable", nil)
		return
	}

	defer r.Body.Close()

	var request registerRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		log.Printf("[http-error] path=%s method=%s msg=invalid register request body err=%v", r.URL.Path, r.Method, err)
		writeJSON(w, http.StatusBadRequest, errcode.HTTPInvalidRequest, "invalid request body", nil)
		return
	}

	result, err := h.playerService.Register(r.Context(), player.RegisterInput{
		AccountName: request.Account,
		Password:    request.Password,
		Gender:      request.Gender,
	})
	if err != nil {
		switch {
		case errors.Is(err, player.ErrInvalidRegisterInput):
			writeJSON(w, http.StatusBadRequest, errcode.HTTPInvalidRequest, "account and password are required", nil)
			return
		case errors.Is(err, player.ErrAccountNameDuplicated), errors.Is(err, player.ErrPlayerNameDuplicated):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "account already exists", nil)
			return
		default:
			log.Printf("[http-error] path=%s method=%s msg=register failed account=%s err=%v", r.URL.Path, r.Method, request.Account, err)
			writeJSON(w, http.StatusInternalServerError, errcode.HTTPInternalServer, "internal server error", nil)
			return
		}
	}

	writeJSON(w, http.StatusOK, errcode.HTTPSuccess, "success", registerResponse{
		PlayerID:   result.PlayerID,
		Account:    result.AccountName,
		PlayerName: result.PlayerName,
		SkinID:     result.SkinID,
	})
}
