package handlers

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type actorPublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPEM string `json:"publicKeyPem"`
}

type actorResponse struct {
	Context           []string       `json:"@context"`
	ID                string         `json:"id"`
	Type              string         `json:"type"`
	PreferredUsername string         `json:"preferredUsername"`
	Name              string         `json:"name"`
	Summary           string         `json:"summary"`
	Inbox             string         `json:"inbox"`
	Outbox            string         `json:"outbox"`
	Followers         string         `json:"followers"`
	Following         string         `json:"following"`
	PublicKey         actorPublicKey `json:"publicKey"`
}

func Actor(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.PathValue("username")
		if username == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid_username",
			})
			return
		}

		var actor actorResponse
		err := deps.PG.QueryRow(r.Context(), `
SELECT
  actor_url,
  username,
  display_name,
  summary,
  inbox_url,
  outbox_url,
  followers_url,
  following_url,
  public_key_pem
FROM actors
WHERE local = TRUE
  AND username = $1
  AND domain = $2
`,
			username,
			deps.Config.AppDomain,
		).Scan(
			&actor.ID,
			&actor.PreferredUsername,
			&actor.Name,
			&actor.Summary,
			&actor.Inbox,
			&actor.Outbox,
			&actor.Followers,
			&actor.Following,
			&actor.PublicKey.PublicKeyPEM,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "actor_not_found",
			})
			return
		}
		if err != nil {
			deps.Logger.Error("actor lookup failed", "error", err, "username", username)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal_server_error",
			})
			return
		}

		actor.Context = []string{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		}
		actor.Type = "Person"
		actor.PublicKey.ID = actor.ID + "#main-key"
		actor.PublicKey.Owner = actor.ID

		writeActivityJSON(w, http.StatusOK, actor)
	})
}
