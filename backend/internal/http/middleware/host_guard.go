package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

func EnforcePublicHost(appEnv, appDomain string) Middleware {
	if !isProdEnv(appEnv) {
		return func(next http.Handler) http.Handler { return next }
	}

	allowedHost := normalizeHost(appDomain)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqHost := normalizeHost(effectiveRequestHost(r))
			if reqHost == "" {
				writeHostGuardError(w, "invalid_host")
				return
			}

			if isLoopbackHost(reqHost) {
				writeHostGuardError(w, "localhost_not_allowed_in_prod")
				return
			}

			if allowedHost != "" && reqHost != allowedHost {
				writeHostGuardError(w, "host_not_allowed")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func effectiveRequestHost(r *http.Request) string {
	if xfh := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); xfh != "" {
		parts := strings.Split(xfh, ",")
		return strings.TrimSpace(parts[0])
	}
	if host := strings.TrimSpace(r.Host); host != "" {
		return host
	}
	return strings.TrimSpace(r.URL.Host)
}

func normalizeHost(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}

	if strings.Contains(raw, "://") {
		// Strip scheme if a full URL is mistakenly passed.
		parts := strings.SplitN(raw, "://", 2)
		raw = parts[1]
	}

	raw = strings.TrimSuffix(raw, ".")

	if strings.Contains(raw, "/") {
		raw = strings.SplitN(raw, "/", 2)[0]
	}

	if strings.Contains(raw, ":") {
		if host, _, err := net.SplitHostPort(raw); err == nil {
			return strings.Trim(strings.TrimSpace(host), "[]")
		}
	}

	return strings.Trim(raw, "[]")
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func writeHostGuardError(w http.ResponseWriter, code string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": code,
	})
}

func isProdEnv(env string) bool {
	normalized := strings.TrimSpace(strings.ToLower(env))
	return normalized == "prod" || normalized == "production"
}
