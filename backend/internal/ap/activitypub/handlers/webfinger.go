package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type webFingerLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
	Href string `json:"href"`
}

type webFingerResponse struct {
	Subject string          `json:"subject"`
	Links   []webFingerLink `json:"links"`
}

func WebFinger(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resource := strings.TrimSpace(r.URL.Query().Get("resource"))
		username, domain, err := parseAcctResource(resource)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid_resource",
			})
			return
		}

		var actorURL string
		err = deps.PG.QueryRow(
			r.Context(),
			`SELECT actor_url FROM actors WHERE local = TRUE AND username = $1 AND domain = $2`,
			username,
			domain,
		).Scan(&actorURL)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "actor_not_found",
			})
			return
		}
		if err != nil {
			deps.Logger.Error("webfinger lookup failed", "error", err, "username", username, "domain", domain)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal_server_error",
			})
			return
		}

		w.Header().Set("Content-Type", "application/jrd+json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(webFingerResponse{
			Subject: "acct:" + username + "@" + domain,
			Links: []webFingerLink{
				{
					Rel:  "self",
					Type: "application/activity+json",
					Href: actorURL,
				},
			},
		})
	})
}

func parseAcctResource(resource string) (username string, domain string, err error) {
	if resource == "" {
		return "", "", errors.New("resource is required")
	}
	if !strings.HasPrefix(resource, "acct:") {
		return "", "", errors.New("resource must begin with acct:")
	}

	parts := strings.SplitN(strings.TrimPrefix(resource, "acct:"), "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("resource must look like acct:user@domain")
	}

	return parts[0], parts[1], nil
}
