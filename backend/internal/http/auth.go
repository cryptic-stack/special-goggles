package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,30}$`)

type registerRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleAuthRegister(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := decodeAuthBody[registerRequest](w, r)
		if err != nil {
			status := http.StatusBadRequest
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				status = http.StatusRequestEntityTooLarge
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}

		username := strings.ToLower(strings.TrimSpace(payload.Username))
		if !usernamePattern.MatchString(username) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_username"})
			return
		}

		email := strings.ToLower(strings.TrimSpace(payload.Email))
		if email == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_required"})
			return
		}
		if _, err := mail.ParseAddress(email); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_email"})
			return
		}

		password := strings.TrimSpace(payload.Password)
		if len(password) < 10 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password_too_short"})
			return
		}

		displayName := strings.TrimSpace(payload.DisplayName)
		if displayName == "" {
			displayName = username
		}
		if len(displayName) > 80 {
			displayName = displayName[:80]
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			deps.Logger.Error("password hash failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			deps.Logger.Error("rsa key generation failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		publicKeyPEM, err := marshalPublicKeyPEM(&privateKey.PublicKey)
		if err != nil {
			deps.Logger.Error("public key marshal failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		privateKeyPEM := marshalPrivateKeyPEM(privateKey)

		tx, err := deps.PG.Begin(r.Context())
		if err != nil {
			deps.Logger.Error("register tx begin failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer tx.Rollback(r.Context())

		actorURLBase := strings.TrimRight(deps.Config.AppBaseURL, "/") + "/users/" + username

		var actorID int64
		err = tx.QueryRow(r.Context(), `
INSERT INTO actors (
  local,
  username,
  domain,
  display_name,
  summary,
  actor_url,
  inbox_url,
  outbox_url,
  followers_url,
  following_url,
  public_key_pem,
  private_key_pem
) VALUES (
  TRUE,
  $1,
  $2,
  $3,
  '',
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10
)
RETURNING id
`,
			username,
			deps.Config.AppDomain,
			displayName,
			actorURLBase,
			actorURLBase+"/inbox",
			actorURLBase+"/outbox",
			actorURLBase+"/followers",
			actorURLBase+"/following",
			publicKeyPEM,
			privateKeyPEM,
		).Scan(&actorID)
		if err != nil {
			if isUniqueViolation(err) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "username_taken"})
				return
			}
			deps.Logger.Error("register actor insert failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		var userID int64
		err = tx.QueryRow(r.Context(), `
INSERT INTO users (actor_id, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id
`,
			actorID,
			email,
			string(passwordHash),
		).Scan(&userID)
		if err != nil {
			if isUniqueViolation(err) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "email_taken"})
				return
			}
			deps.Logger.Error("register user insert failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		sessionToken, expiresAt, err := issueSessionTx(r.Context(), tx, userID)
		if err != nil {
			deps.Logger.Error("register session insert failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			deps.Logger.Error("register tx commit failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		setSessionCookie(w, sessionToken, expiresAt, isSecureCookieEnv(deps.Config.AppEnv))
		writeJSON(w, http.StatusCreated, map[string]any{
			"user_id":   userID,
			"actor_id":  actorID,
			"username":  username,
			"actor_url": actorURLBase,
		})
	}
}

func handleAuthLogin(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := decodeAuthBody[loginRequest](w, r)
		if err != nil {
			status := http.StatusBadRequest
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				status = http.StatusRequestEntityTooLarge
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}

		username := strings.ToLower(strings.TrimSpace(payload.Username))
		email := strings.ToLower(strings.TrimSpace(payload.Email))
		password := strings.TrimSpace(payload.Password)
		if password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password_required"})
			return
		}
		if username == "" && email == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username_or_email_required"})
			return
		}

		var (
			userID       int64
			passwordHash string
			actorID      int64
			actorURL     string
			actorName    string
		)

		query := `
SELECT
  u.id,
  COALESCE(u.password_hash, ''),
  a.id,
  COALESCE(a.actor_url, ''),
  COALESCE(a.username, '')
FROM users u
JOIN actors a ON a.id = u.actor_id
WHERE a.local = TRUE
  AND a.domain = $1
  AND a.username = $2
LIMIT 1
`
		arg := username
		if email != "" {
			query = `
SELECT
  u.id,
  COALESCE(u.password_hash, ''),
  a.id,
  COALESCE(a.actor_url, ''),
  COALESCE(a.username, '')
FROM users u
JOIN actors a ON a.id = u.actor_id
WHERE a.local = TRUE
  AND lower(COALESCE(u.email, '')) = $1
LIMIT 1
`
			arg = email
		}

		if email != "" {
			err = deps.PG.QueryRow(r.Context(), query, arg).Scan(&userID, &passwordHash, &actorID, &actorURL, &actorName)
		} else {
			err = deps.PG.QueryRow(r.Context(), query, deps.Config.AppDomain, arg).Scan(&userID, &passwordHash, &actorID, &actorURL, &actorName)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
			return
		}
		if err != nil {
			deps.Logger.Error("login lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if passwordHash == "" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
			return
		}

		sessionToken, expiresAt, err := issueSession(r.Context(), deps, userID)
		if err != nil {
			deps.Logger.Error("login session issue failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		setSessionCookie(w, sessionToken, expiresAt, isSecureCookieEnv(deps.Config.AppEnv))

		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":   userID,
			"actor_id":  actorID,
			"username":  actorName,
			"actor_url": actorURL,
			"expires":   expiresAt.UTC().Format(http.TimeFormat),
		})
	}
}

func handleAuthLogout(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err == nil {
			sessionID := strings.TrimSpace(cookie.Value)
			if sessionID != "" {
				if _, err := deps.PG.Exec(r.Context(), `DELETE FROM sessions WHERE id = $1`, sessionID); err != nil {
					deps.Logger.Error("logout session delete failed", "error", err)
				}
			}
		}

		clearSessionCookie(w, isSecureCookieEnv(deps.Config.AppEnv))
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAuthMe(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			deps.Logger.Error("auth me session lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"user_id": principal.UserID,
			"actor": map[string]any{
				"id":        principal.ActorID,
				"username":  principal.Username,
				"actor_url": principal.ActorURL,
			},
			"email": principal.Email,
		})
	}
}

func decodeAuthBody[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var zero T
	if r.Body == nil {
		return zero, errors.New("missing request body")
	}

	body := http.MaxBytesReader(w, r.Body, 64<<10)
	defer body.Close()

	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	var payload T
	if err := dec.Decode(&payload); err != nil {
		return zero, errors.New("invalid_json")
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return zero, errors.New("invalid_json")
	}

	return payload, nil
}

func marshalPublicKeyPEM(publicKey *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	})), nil
}

func marshalPrivateKeyPEM(privateKey *rsa.PrivateKey) string {
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}))
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key value")
}

func issueSessionTx(ctx context.Context, tx pgx.Tx, userID int64) (string, time.Time, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(sessionTTL)
	if _, err := tx.Exec(ctx, `
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
