package middleware

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimit(rps float64, burst int) Middleware {
	var (
		mu       sync.Mutex
		visitors = map[string]*visitor{}
	)

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	getVisitor := func(ip string) *visitor {
		mu.Lock()
		defer mu.Unlock()

		v, exists := visitors[ip]
		if !exists {
			adjustedBurst := burst
			if adjustedBurst <= 0 {
				adjustedBurst = int(math.Ceil(rps))
			}
			v = &visitor{
				limiter: rate.NewLimiter(rate.Limit(rps), adjustedBurst),
			}
			visitors[ip] = v
		}

		v.lastSeen = time.Now()
		return v
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r.RemoteAddr)
			if ip != "" {
				if !getVisitor(ip).limiter.Allow() {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusTooManyRequests)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error": "rate_limited",
					})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
