package httptransport

import "net/http"

func NewRouter(loginHandler *LoginHandler, adminHandlers AdminHandlers, wsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/auth/login", loginHandler)
	mux.Handle("/api/admin/auth/login", adminHandlers.Login)
	mux.Handle("/api/admin/me", adminHandlers.Me)
	mux.Handle("/api/admin/healthz", adminHandlers.Health)
	mux.Handle("/api/admin/players", adminHandlers.Players)
	mux.Handle("/api/admin/players/", adminHandlers.Players)
	mux.Handle("/api/admin/pets", adminHandlers.Pets)
	mux.Handle("/api/admin/pets/", adminHandlers.Pets)
	mux.Handle("/api/admin/bags", adminHandlers.Bags)
	mux.Handle("/api/admin/bags/", adminHandlers.Bags)
	mux.Handle("/api/admin/quests", adminHandlers.Quests)
	mux.Handle("/api/admin/quests/", adminHandlers.Quests)
	mux.Handle("/api/admin/npcs", adminHandlers.NPCs)
	mux.Handle("/api/admin/npcs/", adminHandlers.NPCs)
	mux.Handle("/ws", wsHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, 200, "success", map[string]string{"status": "ok"})
	})
	return mux
}
