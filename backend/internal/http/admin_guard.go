package httpapi

import (
	"errors"
	"net/http"
)

func requireAdmin(deps Dependencies, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			deps.Logger.Error("admin principal lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if !deps.Config.IsAdminUsername(principal.Username) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		next(w, r)
	}
}
