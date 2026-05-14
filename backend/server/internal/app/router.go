package app

import (
	"net/http"

	httptransport "pocket-pet-remake/server/internal/transport/http"
	wstransport "pocket-pet-remake/server/internal/transport/ws"
)

func buildHTTPHandler(loginHandler *httptransport.LoginHandler, hub *wstransport.Hub) http.Handler {
	return httptransport.NewRouter(loginHandler, hub)
}
