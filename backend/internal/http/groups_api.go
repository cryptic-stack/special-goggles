package httpapi

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var groupSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

type createGroupRequest struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

func handleCreateGroup(deps Dependencies) http.HandlerFunc {
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

		payload, err := decodeAuthBody[createGroupRequest](w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		slug := strings.ToLower(strings.TrimSpace(payload.Slug))
		if !groupSlugPattern.MatchString(slug) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_slug"})
			return
		}

		title := strings.TrimSpace(payload.Title)
		if title == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title_required"})
			return
		}
		if len(title) > 120 {
			title = title[:120]
		}
		summary := strings.TrimSpace(payload.Summary)
		if len(summary) > 500 {
			summary = summary[:500]
		}

		tx, err := deps.PG.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer tx.Rollback(r.Context())

		var groupID int64
		err = tx.QueryRow(r.Context(), `
INSERT INTO groups (local, slug, title, summary)
VALUES (TRUE, $1, $2, $3)
RETURNING id
`,
			slug,
			title,
			summary,
		).Scan(&groupID)
		if err != nil {
			if isUniqueViolation(err) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "slug_taken"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := tx.Exec(r.Context(), `
INSERT INTO group_memberships (group_id, actor_id, role)
VALUES ($1, $2, 'owner')
`,
			groupID,
			principal.ActorID,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":    groupID,
			"slug":  slug,
			"title": title,
		})
	}
}

func handleJoinGroup(deps Dependencies) http.HandlerFunc {
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

		slug := strings.ToLower(strings.TrimSpace(r.PathValue("slug")))
		if slug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_slug"})
			return
		}

		var groupID int64
		if err := deps.PG.QueryRow(r.Context(), `
SELECT id
FROM groups
WHERE slug = $1
LIMIT 1
`,
			slug,
		).Scan(&groupID); errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "group_not_found"})
			return
		} else if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := deps.PG.Exec(r.Context(), `
INSERT INTO group_memberships (group_id, actor_id, role)
VALUES ($1, $2, 'member')
ON CONFLICT (group_id, actor_id) DO NOTHING
`,
			groupID,
			principal.ActorID,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleCreateGroupPost(deps Dependencies) http.HandlerFunc {
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

		slug := strings.ToLower(strings.TrimSpace(r.PathValue("slug")))
		if slug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_slug"})
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
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer tx.Rollback(r.Context())

		var groupID int64
		if err := tx.QueryRow(r.Context(), `
SELECT g.id
FROM groups g
JOIN group_memberships m ON m.group_id = g.id
WHERE g.slug = $1
  AND m.actor_id = $2
LIMIT 1
`,
			slug,
			principal.ActorID,
		).Scan(&groupID); errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "membership_required"})
			return
		} else if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

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
			principal.ActorID,
			payload.InReplyTo,
			contentHTML,
			contentText,
			visibility,
			payload.Sensitive,
		).Scan(&noteID, &publishedAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		noteURL := strings.TrimRight(deps.Config.AppBaseURL, "/") + "/notes/" + strconv.FormatInt(noteID, 10)
		if _, err := tx.Exec(r.Context(), `UPDATE notes SET note_url = $1 WHERE id = $2`, noteURL, noteID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := tx.Exec(r.Context(), `
INSERT INTO group_posts (group_id, note_id)
VALUES ($1, $2)
`,
			groupID,
			noteID,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := tx.Exec(r.Context(), `
INSERT INTO timeline_items (user_actor_id, note_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`,
			principal.ActorID,
			noteID,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":         noteID,
			"group_slug": slug,
			"note_url":   noteURL,
			"published":  publishedAt.UTC().Format(time.RFC3339),
		})
	}
}

func handleGroupTimeline(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := strings.ToLower(strings.TrimSpace(r.PathValue("slug")))
		if slug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_slug"})
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
FROM group_posts gp
JOIN groups g ON g.id = gp.group_id
JOIN notes n ON n.id = gp.note_id
JOIN actors a ON a.id = n.actor_id
WHERE g.slug = $1
  AND ($2 = 0 OR n.id < $2)
ORDER BY gp.created_at DESC, n.id DESC
LIMIT $3
`,
			slug,
			maxID,
			limit+1,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer rows.Close()

		writeTimelineRows(w, rows, limit)
	}
}
