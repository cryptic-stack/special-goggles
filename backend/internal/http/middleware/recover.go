package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func RecoverJSON(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}

				logger.Error("panic recovered",
					"request_id", FromContext(r.Context()),
					"path", r.URL.Path,
					"method", r.Method,
					"panic", rec,
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "internal_server_error",
				})
			}()

			next.ServeHTTP(w, r)
		})
	}
}
