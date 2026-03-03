package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/cryptic-stack/special-goggles/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Config config.Config
	Logger *slog.Logger
	PG     *pgxpool.Pool
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeActivityJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
