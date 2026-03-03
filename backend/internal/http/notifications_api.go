package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

func handleListNotifications(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		limit := timelineLimit(r.URL.Query().Get("limit"))
		rows, err := deps.PG.Query(r.Context(), `
SELECT
  n.id,
  n.kind,
  COALESCE(n.actor_id, 0),
  COALESCE(a.username, ''),
  COALESCE(a.actor_url, ''),
  COALESCE(n.note_id, 0),
  n.created_at,
  n.read_at IS NOT NULL
FROM notifications n
LEFT JOIN actors a ON a.id = n.actor_id
WHERE n.user_actor_id = $1
ORDER BY n.created_at DESC, n.id DESC
LIMIT $2
`,
			principal.ActorID,
			limit,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer rows.Close()

		items := make([]map[string]any, 0, limit)
		for rows.Next() {
			var (
				id       int64
				kind     string
				actorID  int64
				username string
				actorURL string
				noteID   int64
				created  time.Time
				read     bool
			)
			if err := rows.Scan(&id, &kind, &actorID, &username, &actorURL, &noteID, &created, &read); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}
			item := map[string]any{
				"id":         id,
				"kind":       kind,
				"actor_id":   actorID,
				"username":   username,
				"actor_url":  actorURL,
				"note_id":    noteID,
				"created_at": created.UTC().Format(time.RFC3339),
				"read":       read,
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func handleReadAllNotifications(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := deps.PG.Exec(r.Context(), `
UPDATE notifications
SET read_at = now()
WHERE user_actor_id = $1
  AND read_at IS NULL
`,
			principal.ActorID,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleCreateReport(deps Dependencies) http.HandlerFunc {
	type reportRequest struct {
		TargetActorID int64  `json:"target_actor_id"`
		TargetNoteID  int64  `json:"target_note_id"`
		Reason        string `json:"reason"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		payload, err := decodeAuthBody[reportRequest](w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		reason := payload.Reason
		reason = strings.TrimSpace(reason)
		if len(reason) > 1000 {
			reason = reason[:1000]
		}
		if reason == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reason_required"})
			return
		}

		var reportID int64
		err = deps.PG.QueryRow(r.Context(), `
INSERT INTO reports (reporter_actor_id, target_actor_id, target_note_id, reason)
VALUES ($1, NULLIF($2, 0), NULLIF($3, 0), $4)
RETURNING id
`,
			principal.ActorID,
			payload.TargetActorID,
			payload.TargetNoteID,
			reason,
		).Scan(&reportID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id": reportID,
		})
	}
}

func handleListReports(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := deps.PG.Query(r.Context(), `
SELECT id, reporter_actor_id, COALESCE(target_actor_id, 0), COALESCE(target_note_id, 0), reason, status, created_at, updated_at
FROM reports
ORDER BY created_at DESC, id DESC
LIMIT 100
`)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer rows.Close()

		items := make([]map[string]any, 0, 100)
		for rows.Next() {
			var (
				id, reporterID, targetActorID, targetNoteID int64
				reason, status                              string
				createdAt, updatedAt                        time.Time
			)
			if err := rows.Scan(&id, &reporterID, &targetActorID, &targetNoteID, &reason, &status, &createdAt, &updatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}
			items = append(items, map[string]any{
				"id":              id,
				"reporter_actor":  reporterID,
				"target_actor_id": targetActorID,
				"target_note_id":  targetNoteID,
				"reason":          reason,
				"status":          status,
				"created_at":      createdAt.UTC().Format(time.RFC3339),
				"updated_at":      updatedAt.UTC().Format(time.RFC3339),
			})
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func handleSetDomainPolicy(deps Dependencies) http.HandlerFunc {
	type request struct {
		Domain string `json:"domain"`
		Policy string `json:"policy"`
		Reason string `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := decodeAuthBody[request](w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		domain := strings.ToLower(strings.TrimSpace(payload.Domain))
		policy := strings.ToLower(strings.TrimSpace(payload.Policy))
		reason := strings.TrimSpace(payload.Reason)
		if domain == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain_required"})
			return
		}
		if policy != "allow" && policy != "limit" && policy != "block" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_policy"})
			return
		}

		if _, err := deps.PG.Exec(r.Context(), `
INSERT INTO domain_policies (domain, policy, reason, created_at, updated_at)
VALUES ($1, $2, $3, now(), now())
ON CONFLICT (domain)
DO UPDATE SET
  policy = EXCLUDED.policy,
  reason = EXCLUDED.reason,
  updated_at = now()
`,
			domain, policy, reason,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
