package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"

	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/platform/errcode"
)

type LoginHandler struct {
	authService *auth.Service
}

type loginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
}

type loginResponse struct {
	PlayerID   uint64 `json:"player_id"`
	PlayerName string `json:"player_name"`
	AccessJWT  string `json:"access_jwt"`
	WSToken    string `json:"ws_token"`
	WSExpireAt int64  `json:"ws_expire_at"`
}

func NewLoginHandler(authService *auth.Service) *LoginHandler {
	return &LoginHandler{authService: authService}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errcode.HTTPInvalidRequest, "method not allowed", nil)
		return
	}

	defer r.Body.Close()

	var request loginRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errcode.HTTPInvalidRequest, "invalid request body", nil)
		return
	}

	result, err := h.authService.Login(r.Context(), request.Account, request.Password, request.DeviceID)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, errcode.HTTPUnauthorized, "invalid credentials", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, errcode.HTTPInternalServer, "internal server error", nil)
		return
	}

	writeJSON(w, http.StatusOK, errcode.HTTPSuccess, "success", loginResponse{
		PlayerID:   result.PlayerID,
		PlayerName: result.PlayerName,
		AccessJWT:  result.AccessJWT,
		WSToken:    result.WSToken,
		WSExpireAt: result.WSExpireAt,
	})
}
