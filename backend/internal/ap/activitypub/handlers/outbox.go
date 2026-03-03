package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type outboxActor struct {
	ID           int64
	ActorURL     string
	OutboxURL    string
	FollowersURL string
}

type outboxNote struct {
	ID          int64
	NoteURL     string
	ContentHTML string
	Sensitive   bool
	InReplyTo   string
	PublishedAt time.Time
}

func Outbox(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, err := loadOutboxActor(r, deps)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "actor_not_found"})
			return
		}
		if err != nil {
			deps.Logger.Error("outbox actor lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if r.URL.Query().Get("page") != "true" {
			var total int64
			if err := deps.PG.QueryRow(r.Context(), `SELECT COUNT(1) FROM notes WHERE actor_id = $1 AND local = TRUE`, actor.ID).Scan(&total); err != nil {
				deps.Logger.Error("outbox count failed", "error", err, "actor_id", actor.ID)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}

			writeActivityJSON(w, http.StatusOK, map[string]any{
				"@context":   "https://www.w3.org/ns/activitystreams",
				"id":         actor.OutboxURL,
				"type":       "OrderedCollection",
				"totalItems": total,
				"first":      actor.OutboxURL + "?page=true",
			})
			return
		}

		notes, err := loadOutboxNotes(r, deps, actor.ID)
		if err != nil {
			deps.Logger.Error("outbox notes query failed", "error", err, "actor_id", actor.ID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		items := make([]map[string]any, 0, len(notes))
		for _, note := range notes {
			noteID := note.NoteURL
			if noteID == "" {
				noteID = strings.TrimRight(deps.Config.AppBaseURL, "/") + "/notes/" + strconv.FormatInt(note.ID, 10)
			}

			noteObj := map[string]any{
				"id":           noteID,
				"type":         "Note",
				"attributedTo": actor.ActorURL,
				"content":      note.ContentHTML,
				"to":           []string{"https://www.w3.org/ns/activitystreams#Public"},
				"cc":           []string{actor.FollowersURL},
				"published":    note.PublishedAt.UTC().Format(time.RFC3339),
				"sensitive":    note.Sensitive,
			}
			if note.InReplyTo != "" {
				noteObj["inReplyTo"] = note.InReplyTo
			}

			items = append(items, map[string]any{
				"id":        noteID + "/activities/create",
				"type":      "Create",
				"actor":     actor.ActorURL,
				"published": note.PublishedAt.UTC().Format(time.RFC3339),
				"object":    noteObj,
			})
		}

		writeActivityJSON(w, http.StatusOK, map[string]any{
			"@context":     "https://www.w3.org/ns/activitystreams",
			"id":           actor.OutboxURL + "?page=true",
			"type":         "OrderedCollectionPage",
			"partOf":       actor.OutboxURL,
			"orderedItems": items,
		})
	})
}

func Followers(deps Dependencies) http.Handler {
	return collectionActorLinks(deps, "followers")
}

func Following(deps Dependencies) http.Handler {
	return collectionActorLinks(deps, "following")
}

func collectionActorLinks(deps Dependencies, mode string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, err := loadOutboxActor(r, deps)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "actor_not_found"})
			return
		}
		if err != nil {
			deps.Logger.Error("actor lookup failed", "error", err, "mode", mode)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		collectionID := actor.ActorURL + "/" + mode
		if mode == "followers" {
			collectionID = actor.FollowersURL
		}

		countQuery := `
SELECT COUNT(1)
FROM follows f
WHERE f.following_id = $1
  AND f.state = 'accepted'
`
		listQuery := `
SELECT a.actor_url
FROM follows f
JOIN actors a ON a.id = f.follower_id
WHERE f.following_id = $1
  AND f.state = 'accepted'
ORDER BY f.created_at DESC
`

		if mode == "following" {
			countQuery = `
SELECT COUNT(1)
FROM follows f
WHERE f.follower_id = $1
  AND f.state = 'accepted'
`
			listQuery = `
SELECT a.actor_url
FROM follows f
JOIN actors a ON a.id = f.following_id
WHERE f.follower_id = $1
  AND f.state = 'accepted'
ORDER BY f.created_at DESC
`
		}

		if r.URL.Query().Get("page") != "true" {
			var total int64
			if err := deps.PG.QueryRow(r.Context(), countQuery, actor.ID).Scan(&total); err != nil {
				deps.Logger.Error("actor collection count failed", "error", err, "mode", mode)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}

			writeActivityJSON(w, http.StatusOK, map[string]any{
				"@context":   "https://www.w3.org/ns/activitystreams",
				"id":         collectionID,
				"type":       "OrderedCollection",
				"totalItems": total,
				"first":      collectionID + "?page=true",
			})
			return
		}

		rows, err := deps.PG.Query(r.Context(), listQuery, actor.ID)
		if err != nil {
			deps.Logger.Error("actor collection query failed", "error", err, "mode", mode)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer rows.Close()

		items := make([]string, 0)
		for rows.Next() {
			var actorURL string
			if err := rows.Scan(&actorURL); err != nil {
				deps.Logger.Error("actor collection scan failed", "error", err, "mode", mode)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}
			if actorURL != "" {
				items = append(items, actorURL)
			}
		}
		if err := rows.Err(); err != nil {
			deps.Logger.Error("actor collection rows failed", "error", err, "mode", mode)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeActivityJSON(w, http.StatusOK, map[string]any{
			"@context":     "https://www.w3.org/ns/activitystreams",
			"id":           collectionID + "?page=true",
			"type":         "OrderedCollectionPage",
			"partOf":       collectionID,
			"orderedItems": items,
		})
	})
}

func loadOutboxActor(r *http.Request, deps Dependencies) (outboxActor, error) {
	var actor outboxActor
	err := deps.PG.QueryRow(r.Context(), `
SELECT id, actor_url, outbox_url, followers_url
FROM actors
WHERE local = TRUE
  AND username = $1
  AND domain = $2
`,
		strings.TrimSpace(r.PathValue("username")),
		deps.Config.AppDomain,
	).Scan(&actor.ID, &actor.ActorURL, &actor.OutboxURL, &actor.FollowersURL)
	return actor, err
}

func loadOutboxNotes(r *http.Request, deps Dependencies, actorID int64) ([]outboxNote, error) {
	rows, err := deps.PG.Query(r.Context(), `
SELECT id, COALESCE(note_url, ''), content_html, sensitive, COALESCE(in_reply_to_url, ''), published_at
FROM notes
WHERE actor_id = $1
  AND local = TRUE
ORDER BY published_at DESC, id DESC
LIMIT 20
`,
		actorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []outboxNote
	for rows.Next() {
		var n outboxNote
		if err := rows.Scan(&n.ID, &n.NoteURL, &n.ContentHTML, &n.Sensitive, &n.InReplyTo, &n.PublishedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return notes, nil
}
