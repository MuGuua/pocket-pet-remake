package httptransport

import "net/http"

func NewRouter(loginHandler *LoginHandler, wsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/auth/login", loginHandler)
	mux.Handle("/ws", wsHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, 200, "success", map[string]string{"status": "ok"})
	})
	return mux
}
