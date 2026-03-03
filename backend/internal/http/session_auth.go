package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	sessionCookieName = "sg_session"
	sessionTTL        = 30 * 24 * time.Hour
)

var errUnauthorized = errors.New("unauthorized")

type sessionPrincipal struct {
	SessionID string
	UserID    int64
	ActorID   int64
	Username  string
	ActorURL  string
	Email     string
}

func loadSessionPrincipal(ctx context.Context, deps Dependencies, r *http.Request) (sessionPrincipal, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return sessionPrincipal{}, errUnauthorized
	}
	sessionID := strings.TrimSpace(cookie.Value)
	if sessionID == "" {
		return sessionPrincipal{}, errUnauthorized
	}

	var principal sessionPrincipal
	err = deps.PG.QueryRow(ctx, `
SELECT
  s.id,
  u.id,
  a.id,
  a.username,
  COALESCE(a.actor_url, ''),
  COALESCE(u.email, '')
FROM sessions s
JOIN users u ON u.id = s.user_id
JOIN actors a ON a.id = u.actor_id
WHERE s.id = $1
  AND s.expires_at > now()
  AND a.local = TRUE
LIMIT 1
`,
		sessionID,
	).Scan(
		&principal.SessionID,
		&principal.UserID,
		&principal.ActorID,
		&principal.Username,
		&principal.ActorURL,
		&principal.Email,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sessionPrincipal{}, errUnauthorized
	}
	if err != nil {
		return sessionPrincipal{}, err
	}
	return principal, nil
}

func issueSession(ctx context.Context, deps Dependencies, userID int64) (string, time.Time, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(sessionTTL)

	if _, err := deps.PG.Exec(ctx, `
INSERT INTO sessions (id, user_id, expires_at)
VALUES ($1, $2, $3)
`,
		token,
		userID,
		expiresAt,
	); err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func randomToken(byteLen int) (string, error) {
	raw := make([]byte, byteLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func isSecureCookieEnv(env string) bool {
	env = strings.TrimSpace(strings.ToLower(env))
	return env == "prod" || env == "production"
}
