package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type quotePostRequest struct {
	Content       string  `json:"content"`
	Visibility    string  `json:"visibility"`
	Sensitive     bool    `json:"sensitive"`
	AttachmentIDs []int64 `json:"attachment_ids"`
}

type relationRequest struct {
	Target string `json:"target"`
}

func handleCreateQuotePost(deps Dependencies) http.HandlerFunc {
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

		noteID, ok := noteIDFromPath(r)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_note_id"})
			return
		}
		target, err := loadNoteTarget(r.Context(), deps, noteID)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "note_not_found"})
			return
		}
		if err != nil {
			deps.Logger.Error("quote target lookup failed", "error", err, "note_id", noteID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		payload, err := decodeAuthBody[quotePostRequest](w, r)
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

		attachmentIDs, err := normalizeAttachmentIDs(payload.AttachmentIDs)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		actor, err := resolveLocalActorByID(r, deps, principal.ActorID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		tx, err := deps.PG.Begin(r.Context())
		if err != nil {
			deps.Logger.Error("begin quote post tx failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer tx.Rollback(r.Context())

		var createdNoteID int64
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
  sensitive,
  quote_note_id
) VALUES (
  TRUE,
  NULL,
  $1,
  NULL,
  $2,
  $3,
  $4,
  $5,
  $6
)
RETURNING id, published_at
`,
			actor.ID,
			contentHTML,
			contentText,
			visibility,
			payload.Sensitive,
			target.ID,
		).Scan(&createdNoteID, &publishedAt)
		if err != nil {
			deps.Logger.Error("insert quote note failed", "error", err, "actor_id", actor.ID, "target_note_id", target.ID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		noteURL := strings.TrimRight(deps.Config.AppBaseURL, "/") + "/notes/" + strconv.FormatInt(createdNoteID, 10)
		if _, err := tx.Exec(r.Context(), `UPDATE notes SET note_url = $1 WHERE id = $2`, noteURL, createdNoteID); err != nil {
			deps.Logger.Error("update quote note url failed", "error", err, "note_id", createdNoteID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		createActivityID := strings.TrimRight(actor.ActorURL, "/") + "/activities/create/" + strconv.FormatInt(createdNoteID, 10)
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
				"quoteUrl":     target.NoteURL,
			},
		}

		if len(attachmentIDs) > 0 {
			attachments, err := loadOwnedAttachments(r.Context(), tx, actor.ID, attachmentIDs, deps.Config.AppBaseURL)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}

			if len(attachments) != len(attachmentIDs) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "attachment_not_found_or_not_owned"})
				return
			}

			for _, attachmentID := range attachmentIDs {
				if _, err := tx.Exec(r.Context(), `
INSERT INTO note_attachments (note_id, attachment_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`,
					createdNoteID,
					attachmentID,
				); err != nil {
					deps.Logger.Error("insert quote note attachment failed", "error", err, "note_id", createdNoteID, "attachment_id", attachmentID)
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
					return
				}
			}

			if createActivityObject, ok := createActivity["object"].(map[string]any); ok {
				createActivityObject["attachment"] = attachments
			}
		}

		if _, err := tx.Exec(r.Context(), `
INSERT INTO timeline_items (user_actor_id, note_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`,
			actor.ID,
			createdNoteID,
		); err != nil {
			deps.Logger.Error("insert quote author timeline item failed", "error", err, "note_id", createdNoteID)
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
			deps.Logger.Error("query quote followers failed", "error", err, "actor_id", actor.ID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		type followerTarget struct {
			id       int64
			local    bool
			inboxURL string
			actorURL string
		}
		followers := make([]followerTarget, 0, 16)
		for rows.Next() {
			var followerID int64
			var followerLocal bool
			var inboxURL string
			var followerActorURL string
			if err := rows.Scan(&followerID, &followerLocal, &inboxURL, &followerActorURL); err != nil {
				rows.Close()
				deps.Logger.Error("scan quote follower failed", "error", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}
			followers = append(followers, followerTarget{
				id:       followerID,
				local:    followerLocal,
				inboxURL: inboxURL,
				actorURL: followerActorURL,
			})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			deps.Logger.Error("iterate quote followers failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		activityJSON, err := json.Marshal(createActivity)
		if err != nil {
			deps.Logger.Error("marshal quote create activity failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		for _, follower := range followers {
			if follower.local {
				if _, err := tx.Exec(r.Context(), `
INSERT INTO timeline_items (user_actor_id, note_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`,
					follower.id,
					createdNoteID,
				); err != nil {
					deps.Logger.Error("insert quote follower timeline item failed", "error", err, "follower_id", follower.id, "note_id", createdNoteID)
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
					return
				}
				continue
			}
			targetInbox := strings.TrimSpace(follower.inboxURL)
			if targetInbox == "" && follower.actorURL != "" {
				targetInbox = strings.TrimRight(follower.actorURL, "/") + "/inbox"
			}
			if targetInbox == "" {
				continue
			}
			if _, err := tx.Exec(r.Context(), `
INSERT INTO deliveries (target_inbox, activity_id, activity_json)
VALUES ($1, $2, $3)
`,
				targetInbox,
				createActivityID,
				activityJSON,
			); err != nil {
				deps.Logger.Error("enqueue quote remote delivery failed", "error", err, "target_inbox", targetInbox)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}
		}

		if err := tx.Commit(r.Context()); err != nil {
			deps.Logger.Error("commit quote post tx failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":            createdNoteID,
			"note_url":      noteURL,
			"activity_id":   createActivityID,
			"published":     publishedAt.UTC().Format(time.RFC3339),
			"attachments":   attachmentIDs,
			"quote_note_id": target.ID,
		})
	}
}

func handleCreateBookmark(deps Dependencies) http.HandlerFunc {
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

		noteID, ok := noteIDFromPath(r)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_note_id"})
			return
		}

		if _, err := loadNoteTarget(r.Context(), deps, noteID); errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "note_not_found"})
			return
		} else if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := deps.PG.Exec(r.Context(), `
INSERT INTO bookmarks (actor_id, note_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`,
			principal.ActorID,
			noteID,
		); err != nil {
			deps.Logger.Error("insert bookmark failed", "error", err, "actor_id", principal.ActorID, "note_id", noteID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"note_id": noteID})
	}
}

func handleDeleteBookmark(deps Dependencies) http.HandlerFunc {
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

		noteID, ok := noteIDFromPath(r)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_note_id"})
			return
		}

		if _, err := deps.PG.Exec(r.Context(), `
DELETE FROM bookmarks
WHERE actor_id = $1
  AND note_id = $2
`,
			principal.ActorID,
			noteID,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleListBookmarks(deps Dependencies) http.HandlerFunc {
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
FROM bookmarks b
JOIN notes n ON n.id = b.note_id
JOIN actors a ON a.id = n.actor_id
WHERE b.actor_id = $1
  AND ($2 = 0 OR n.id < $2)
  AND NOT EXISTS (
    SELECT 1
    FROM blocks bl
    WHERE (bl.actor_id = $1 AND bl.target_actor_id = n.actor_id)
       OR (bl.actor_id = n.actor_id AND bl.target_actor_id = $1)
  )
  AND NOT EXISTS (
    SELECT 1
    FROM mutes m
    WHERE m.actor_id = $1
      AND m.target_actor_id = n.actor_id
  )
ORDER BY b.created_at DESC, n.id DESC
LIMIT $3
`,
			principal.ActorID,
			maxID,
			limit+1,
		)
		if err != nil {
			deps.Logger.Error("query bookmarks failed", "error", err, "actor_id", principal.ActorID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer rows.Close()

		writeTimelineRows(r.Context(), w, deps, rows, limit)
	}
}

func handleCreateMute(deps Dependencies) http.HandlerFunc {
	return handleCreateActorRelation(deps, "mutes")
}

func handleDeleteMute(deps Dependencies) http.HandlerFunc {
	return handleDeleteActorRelation(deps, "mutes")
}

func handleListMutes(deps Dependencies) http.HandlerFunc {
	return handleListActorRelations(deps, "mutes")
}

func handleCreateBlock(deps Dependencies) http.HandlerFunc {
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

		payload, err := decodeAuthBody[relationRequest](w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		target, err := resolveFollowTarget(r.Context(), deps, payload.Target, true)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if target.ID == principal.ActorID {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot_block_self"})
			return
		}

		tx, err := deps.PG.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer tx.Rollback(r.Context())

		if _, err := tx.Exec(r.Context(), `
INSERT INTO blocks (actor_id, target_actor_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`,
			principal.ActorID,
			target.ID,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := tx.Exec(r.Context(), `
DELETE FROM follows
WHERE (follower_id = $1 AND following_id = $2)
   OR (follower_id = $2 AND following_id = $1)
`,
			principal.ActorID,
			target.ID,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := tx.Exec(r.Context(), `
DELETE FROM timeline_items ti
USING notes n
WHERE ti.note_id = n.id
  AND ti.user_actor_id = $1
  AND n.actor_id = $2
`,
			principal.ActorID,
			target.ID,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"target_id":   target.ID,
			"target_user": target.Username,
			"target":      target.ActorURL,
		})
	}
}

func handleDeleteBlock(deps Dependencies) http.HandlerFunc {
	return handleDeleteActorRelation(deps, "blocks")
}

func handleListBlocks(deps Dependencies) http.HandlerFunc {
	return handleListActorRelations(deps, "blocks")
}

func handleCreateActorRelation(deps Dependencies, table string) http.HandlerFunc {
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

		payload, err := decodeAuthBody[relationRequest](w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		target, err := resolveFollowTarget(r.Context(), deps, payload.Target, true)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if target.ID == principal.ActorID {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot_target_self"})
			return
		}

		query := "INSERT INTO " + table + " (actor_id, target_actor_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
		if _, err := deps.PG.Exec(r.Context(), query, principal.ActorID, target.ID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"target_id":   target.ID,
			"target_user": target.Username,
			"target":      target.ActorURL,
		})
	}
}

func handleDeleteActorRelation(deps Dependencies, table string) http.HandlerFunc {
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

		payload, err := decodeAuthBody[relationRequest](w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		target, err := resolveFollowTarget(r.Context(), deps, payload.Target, false)
		if errors.Is(err, pgx.ErrNoRows) || (err != nil && err.Error() == "target_not_found") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		query := "DELETE FROM " + table + " WHERE actor_id = $1 AND target_actor_id = $2"
		if _, err := deps.PG.Exec(r.Context(), query, principal.ActorID, target.ID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleListActorRelations(deps Dependencies, table string) http.HandlerFunc {
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

		rows, err := deps.PG.Query(r.Context(), "\nSELECT a.id, COALESCE(a.username, ''), COALESCE(a.actor_url, ''), r.created_at\nFROM "+table+" r\nJOIN actors a ON a.id = r.target_actor_id\nWHERE r.actor_id = $1\nORDER BY r.created_at DESC\nLIMIT 100\n", principal.ActorID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer rows.Close()

		items := make([]map[string]any, 0, 16)
		for rows.Next() {
			var actorID int64
			var username string
			var actorURL string
			var createdAt time.Time
			if err := rows.Scan(&actorID, &username, &actorURL, &createdAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}
			items = append(items, map[string]any{
				"actor_id":   actorID,
				"username":   username,
				"actor_url":  actorURL,
				"created_at": createdAt.UTC().Format(time.RFC3339),
			})
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}
