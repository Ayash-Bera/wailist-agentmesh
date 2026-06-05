package api

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/agentmesh/backend/internal/api/handlers"
)

func corsMiddleware(next http.Handler) http.Handler {
	origin := os.Getenv("CORS_ORIGIN")
	if origin == "" {
		origin = "*"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authMiddleware always sets userID="dev" (Phase 1 stub).
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		ctx := context.WithValue(r.Context(), handlers.CtxUserID, "dev")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
