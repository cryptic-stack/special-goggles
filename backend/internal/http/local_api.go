package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var stripHTMLPattern = regexp.MustCompile(`<[^>]*>`)

type localActor struct {
	ID           int64
	Username     string
	ActorURL     string
	FollowersURL string
}

type createPostRequest struct {
	Content    string `json:"content"`
	InReplyTo  string `json:"in_reply_to"`
	Visibility string `json:"visibility"`
	Sensitive  bool   `json:"sensitive"`
}

func handleCreatePost(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			deps.Logger.Error("post session lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		actor, err := resolveLocalActorByID(r, deps, principal.ActorID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		payload, err := decodeCreatePostRequest(w, r)
		if err != nil {
			status := http.StatusBadRequest
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				status = http.StatusRequestEntityTooLarge
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}

		contentHTML := strings.TrimSpace(payload.Content)
		if contentHTML == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
			return
		}
		contentText := strings.TrimSpace(stripHTMLPattern.ReplaceAllString(contentHTML, ""))
		if contentText == "" {
			contentText = contentHTML
		}

		visibility := strings.TrimSpace(strings.ToLower(payload.Visibility))
		if visibility == "" {
			visibility = "public"
		}
		if visibility != "public" && visibility != "unlisted" && visibility != "followers" && visibility != "direct" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid visibility"})
			return
		}

		tx, err := deps.PG.Begin(r.Context())
		if err != nil {
			deps.Logger.Error("begin create post tx failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer tx.Rollback(r.Context())

		var noteID int64
		var publishedAt time.Time
		err = tx.QueryRow(r.Context(), `
INSERT INTO notes (
  local,
  note_url,
  actor_id,
  in_reply_to_url,
  content_html,
  content_text,
  visibility,
  sensitive
) VALUES (
  TRUE,
  NULL,
  $1,
  NULLIF($2, ''),
  $3,
  $4,
  $5,
  $6
)
RETURNING id, published_at
`,
			actor.ID,
			payload.InReplyTo,
			contentHTML,
			contentText,
			visibility,
			payload.Sensitive,
		).Scan(&noteID, &publishedAt)
		if err != nil {
			deps.Logger.Error("insert note failed", "error", err, "actor_id", actor.ID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		noteURL := strings.TrimRight(deps.Config.AppBaseURL, "/") + "/notes/" + strconv.FormatInt(noteID, 10)
		if _, err := tx.Exec(r.Context(), `UPDATE notes SET note_url = $1 WHERE id = $2`, noteURL, noteID); err != nil {
			deps.Logger.Error("update note url failed", "error", err, "note_id", noteID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		createActivityID := strings.TrimRight(actor.ActorURL, "/") + "/activities/create/" + strconv.FormatInt(noteID, 10)
		createActivity := map[string]any{
			"@context": "https://www.w3.org/ns/activitystreams",
			"id":       createActivityID,
			"type":     "Create",
			"actor":    actor.ActorURL,
			"to":       []string{"https://www.w3.org/ns/activitystreams#Public"},
			"object": map[string]any{
				"id":           noteURL,
				"type":         "Note",
				"attributedTo": actor.ActorURL,
				"content":      contentHTML,
				"published":    publishedAt.UTC().Format(time.RFC3339),
				"sensitive":    payload.Sensitive,
				"to":           []string{"https://www.w3.org/ns/activitystreams#Public"},
				"cc":           []string{actor.FollowersURL},
			},
		}

		if _, err := tx.Exec(r.Context(), `
INSERT INTO timeline_items (user_actor_id, note_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`,
			actor.ID,
			noteID,
		); err != nil {
			deps.Logger.Error("insert author timeline item failed", "error", err, "note_id", noteID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		rows, err := tx.Query(r.Context(), `
SELECT a.id, a.local, COALESCE(a.inbox_url, ''), COALESCE(a.actor_url, '')
FROM follows f
JOIN actors a ON a.id = f.follower_id
WHERE f.following_id = $1
  AND f.state = 'accepted'
`,
			actor.ID,
		)
		if err != nil {
			deps.Logger.Error("query followers failed", "error", err, "actor_id", actor.ID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		for rows.Next() {
			var followerID int64
			var followerLocal bool
			var inboxURL string
			var followerActorURL string
			if err := rows.Scan(&followerID, &followerLocal, &inboxURL, &followerActorURL); err != nil {
				rows.Close()
				deps.Logger.Error("scan follower failed", "error", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}

			if followerLocal {
				if _, err := tx.Exec(r.Context(), `
INSERT INTO timeline_items (user_actor_id, note_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`,
					followerID, noteID,
				); err != nil {
					rows.Close()
					deps.Logger.Error("insert follower timeline item failed", "error", err, "follower_id", followerID, "note_id", noteID)
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
					return
				}
				continue
			}

			targetInbox := strings.TrimSpace(inboxURL)
			if targetInbox == "" && followerActorURL != "" {
				targetInbox = strings.TrimRight(followerActorURL, "/") + "/inbox"
			}
			if targetInbox == "" {
				continue
			}

			activityJSON, err := json.Marshal(createActivity)
			if err != nil {
				rows.Close()
				deps.Logger.Error("marshal create activity failed", "error", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}

			if _, err := tx.Exec(r.Context(), `
INSERT INTO deliveries (target_inbox, activity_id, activity_json)
VALUES ($1, $2, $3)
`,
				targetInbox,
				createActivityID,
				activityJSON,
			); err != nil {
				rows.Close()
				deps.Logger.Error("enqueue remote delivery failed", "error", err, "target_inbox", targetInbox)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			deps.Logger.Error("iterate followers failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			deps.Logger.Error("commit create post tx failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":          noteID,
			"note_url":    noteURL,
			"activity_id": createActivityID,
			"published":   publishedAt.UTC().Format(time.RFC3339),
		})
	}
}

func handleHomeTimeline(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, err := resolveLocalActor(r, deps)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		limit := timelineLimit(r.URL.Query().Get("limit"))
		maxID := timelineMaxID(r.URL.Query().Get("max_id"))
		rows, err := deps.PG.Query(r.Context(), `
SELECT
  n.id,
  COALESCE(n.note_url, ''),
  n.content_html,
  n.content_text,
  n.published_at,
  COALESCE(a.actor_url, ''),
  COALESCE(a.username, '')
FROM timeline_items t
JOIN notes n ON n.id = t.note_id
JOIN actors a ON a.id = n.actor_id
WHERE t.user_actor_id = $1
  AND ($2 = 0 OR n.id < $2)
ORDER BY t.created_at DESC, n.id DESC
LIMIT $3
`,
			actor.ID, maxID, limit+1,
		)
		if err != nil {
			deps.Logger.Error("query home timeline failed", "error", err, "actor_id", actor.ID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer rows.Close()

		writeTimelineRows(w, rows, limit)
	}
}

func handleLocalTimeline(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := timelineLimit(r.URL.Query().Get("limit"))
		maxID := timelineMaxID(r.URL.Query().Get("max_id"))
		rows, err := deps.PG.Query(r.Context(), `
SELECT
  n.id,
  COALESCE(n.note_url, ''),
  n.content_html,
  n.content_text,
  n.published_at,
  COALESCE(a.actor_url, ''),
  COALESCE(a.username, '')
FROM notes n
JOIN actors a ON a.id = n.actor_id
WHERE n.local = TRUE
  AND ($1 = 0 OR n.id < $1)
ORDER BY n.published_at DESC, n.id DESC
LIMIT $2
`,
			maxID,
			limit+1,
		)
		if err != nil {
			deps.Logger.Error("query local timeline failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer rows.Close()

		writeTimelineRows(w, rows, limit)
	}
}

func decodeCreatePostRequest(w http.ResponseWriter, r *http.Request) (createPostRequest, error) {
	if r.Body == nil {
		return createPostRequest{}, errors.New("missing request body")
	}
	body := http.MaxBytesReader(w, r.Body, 64<<10)
	defer body.Close()

	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	var payload createPostRequest
	if err := dec.Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return createPostRequest{}, errors.New("missing request body")
		}
		return createPostRequest{}, errors.New("invalid_json")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return createPostRequest{}, errors.New("invalid_json")
	}
	return payload, nil
}

func resolveLocalActor(r *http.Request, deps Dependencies) (localActor, error) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		username = "alice"
	}

	var actor localActor
	err := deps.PG.QueryRow(r.Context(), `
SELECT id, username, actor_url, followers_url
FROM actors
WHERE local = TRUE
  AND username = $1
  AND domain = $2
`,
		username, deps.Config.AppDomain,
	).Scan(&actor.ID, &actor.Username, &actor.ActorURL, &actor.FollowersURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return localActor{}, errors.New("actor_not_found")
	}
	if err != nil {
		return localActor{}, errors.New("actor_not_found")
	}
	return actor, nil
}

func resolveLocalActorByID(r *http.Request, deps Dependencies, actorID int64) (localActor, error) {
	var actor localActor
	err := deps.PG.QueryRow(r.Context(), `
SELECT id, username, actor_url, followers_url
FROM actors
WHERE local = TRUE
  AND id = $1
`,
		actorID,
	).Scan(&actor.ID, &actor.Username, &actor.ActorURL, &actor.FollowersURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return localActor{}, errors.New("actor_not_found")
	}
	if err != nil {
		return localActor{}, errors.New("actor_not_found")
	}
	return actor, nil
}

func timelineLimit(raw string) int {
	if raw == "" {
		return 20
	}
	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 20
	}
	if limit <= 0 {
		return 20
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func timelineMaxID(raw string) int64 {
	if raw == "" {
		return 0
	}
	maxID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || maxID <= 0 {
		return 0
	}
	return maxID
}

func writeTimelineRows(w http.ResponseWriter, rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}, limit int) {
	items := make([]map[string]any, 0, limit)
	hasMore := false
	var nextMaxID int64
	for rows.Next() {
		var (
			noteID      int64
			noteURL     string
			contentHTML string
			contentText string
			published   time.Time
			actorURL    string
			username    string
		)
		if err := rows.Scan(&noteID, &noteURL, &contentHTML, &contentText, &published, &actorURL, &username); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if len(items) == limit {
			hasMore = true
			break
		}

		items = append(items, map[string]any{
			"id":           noteID,
			"note_url":     noteURL,
			"content_html": contentHTML,
			"content_text": contentText,
			"published_at": published.UTC().Format(time.RFC3339),
			"actor_url":    actorURL,
			"username":     username,
		})
		nextMaxID = noteID
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
		return
	}

	payload := map[string]any{
		"items":    items,
		"has_more": hasMore,
	}
	if hasMore && nextMaxID > 0 {
		payload["next_max_id"] = nextMaxID
	}
	writeJSON(w, http.StatusOK, payload)
}
