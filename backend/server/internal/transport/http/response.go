package httptransport

import (
	"encoding/json"
	"net/http"
	"time"

	"pocket-pet-remake/server/internal/platform/idgen"
)

type envelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	UUID string `json:"uuid"`
	Data any    `json:"data"`
}

func writeJSON(w http.ResponseWriter, httpStatus, code int, msg string, data any) {
	traceID, err := idgen.RandomHex(16)
	if err != nil {
		traceID = time.Now().UTC().Format("20060102150405")
	}

	payload := envelope{
		Code: code,
		Msg:  msg,
		UUID: traceID,
		Data: data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(payload)
}
