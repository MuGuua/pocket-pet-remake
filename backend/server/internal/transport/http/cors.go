package httptransport

import (
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

// withLocalWebCORS permits Godot Web exports served from localhost or a LAN
// address to call the API on a different port during development.
func withLocalWebCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isLocalWebOrigin(origin) {
			header := w.Header()
			header.Set("Access-Control-Allow-Origin", origin)
			header.Set("Vary", "Origin")
			header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			header.Set("Access-Control-Max-Age", "600")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isLocalWebOrigin only opens CORS for local development origins. Production
// domains should be added through an explicit configuration entry later.
func isLocalWebOrigin(origin string) bool {
	if strings.TrimSpace(origin) == "" {
		return false
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	host := parsed.Hostname()
	if host == "localhost" {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		addr, ok := netip.AddrFromSlice(ip)
		if !ok {
			return false
		}
		return addr.IsLoopback() || addr.IsPrivate()
	}

	return false
}
