package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cryptic-stack/special-goggles/backend/internal/ap/fetch"
	"github.com/cryptic-stack/special-goggles/backend/internal/domain/notifications"
	"github.com/jackc/pgx/v5"
)

type followRequest struct {
	Target string `json:"target"`
}

type actorTarget struct {
	ID       int64
	Local    bool
	Username string
	ActorURL string
	InboxURL string
}

type noteTarget struct {
	ID          int64
	NoteURL     string
	ActorID     int64
	ActorLocal  bool
	ActorURL    string
	ActorInbox  string
	PublishedAt time.Time
}

func handleFollow(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			deps.Logger.Error("follow session lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		payload, err := decodeAuthBody[followRequest](w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		targetRaw := strings.TrimSpace(payload.Target)
		if targetRaw == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "target_required"})
			return
		}

		target, err := resolveFollowTarget(r.Context(), deps, targetRaw, true)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if target.ID == principal.ActorID {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot_follow_self"})
			return
		}

		if target.Local {
			_, err := deps.PG.Exec(r.Context(), `
INSERT INTO follows (follower_id, following_id, state, follow_activity_url)
VALUES ($1, $2, 'accepted', $3)
ON CONFLICT (follower_id, following_id)
DO UPDATE SET state = 'accepted'
`,
				principal.ActorID,
				target.ID,
				"",
			)
			if err != nil {
				deps.Logger.Error("insert local follow failed", "error", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
				return
			}
			actorID := principal.ActorID
			if err := notifications.Insert(r.Context(), deps.PG, target.ID, "follow", &actorID, nil); err != nil {
				deps.Logger.Warn("insert follow notification failed", "error", err)
			}
			writeJSON(w, http.StatusCreated, map[string]any{
				"state":       "accepted",
				"target":      target.ActorURL,
				"federated":   false,
				"target_id":   target.ID,
				"target_user": target.Username,
			})
			return
		}

		targetInbox := strings.TrimSpace(target.InboxURL)
		if targetInbox == "" {
			targetInbox = strings.TrimRight(target.ActorURL, "/") + "/inbox"
		}
		if targetInbox == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "remote_inbox_missing"})
			return
		}

		token, err := randomToken(12)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		followActivityID := strings.TrimRight(principal.ActorURL, "/") + "/activities/follow/" + token
		followActivity := map[string]any{
			"@context": "https://www.w3.org/ns/activitystreams",
			"id":       followActivityID,
			"type":     "Follow",
			"actor":    principal.ActorURL,
			"object":   target.ActorURL,
		}

		activityJSON, err := json.Marshal(followActivity)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		tx, err := deps.PG.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		defer tx.Rollback(r.Context())

		if _, err := tx.Exec(r.Context(), `
INSERT INTO follows (follower_id, following_id, state, follow_activity_url)
VALUES ($1, $2, 'pending', $3)
ON CONFLICT (follower_id, following_id)
DO UPDATE SET
  state = 'pending',
  follow_activity_url = EXCLUDED.follow_activity_url
`,
			principal.ActorID,
			target.ID,
			followActivityID,
		); err != nil {
			deps.Logger.Error("insert remote follow failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := tx.Exec(r.Context(), `
INSERT INTO deliveries (target_inbox, activity_id, activity_json)
VALUES ($1, $2, $3)
`,
			targetInbox,
			followActivityID,
			activityJSON,
		); err != nil {
			deps.Logger.Error("enqueue follow delivery failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			deps.Logger.Error("commit follow tx failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"state":       "pending",
			"target":      target.ActorURL,
			"federated":   true,
			"activity_id": followActivityID,
		})
	}
}

func handleUnfollow(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			deps.Logger.Error("unfollow session lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		payload, err := decodeAuthBody[followRequest](w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		target, err := resolveFollowTarget(r.Context(), deps, payload.Target, false)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var followActivityURL string
		err = deps.PG.QueryRow(r.Context(), `
SELECT COALESCE(follow_activity_url, '')
FROM follows
WHERE follower_id = $1
  AND following_id = $2
`,
			principal.ActorID,
			target.ID,
		).Scan(&followActivityURL)
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err != nil {
			deps.Logger.Error("lookup unfollow relation failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := deps.PG.Exec(r.Context(), `
DELETE FROM follows
WHERE follower_id = $1
  AND following_id = $2
`,
			principal.ActorID,
			target.ID,
		); err != nil {
			deps.Logger.Error("delete follow relation failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if !target.Local && followActivityURL != "" {
			targetInbox := strings.TrimSpace(target.InboxURL)
			if targetInbox == "" {
				targetInbox = strings.TrimRight(target.ActorURL, "/") + "/inbox"
			}
			if targetInbox != "" {
				token, err := randomToken(12)
				if err == nil {
					undoID := strings.TrimRight(principal.ActorURL, "/") + "/activities/undo/" + token
					undo := map[string]any{
						"@context": "https://www.w3.org/ns/activitystreams",
						"id":       undoID,
						"type":     "Undo",
						"actor":    principal.ActorURL,
						"object": map[string]any{
							"id":   followActivityURL,
							"type": "Follow",
						},
					}
					if raw, err := json.Marshal(undo); err == nil {
						if _, err := deps.PG.Exec(r.Context(), `
INSERT INTO deliveries (target_inbox, activity_id, activity_json)
VALUES ($1, $2, $3)
`,
							targetInbox,
							undoID,
							raw,
						); err != nil {
							deps.Logger.Warn("enqueue unfollow undo failed", "error", err)
						}
					}
				}
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleCreateReaction(deps Dependencies, kind, activityType string) http.HandlerFunc {
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
			deps.Logger.Error("reaction note lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		token, err := randomToken(12)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		activityID := strings.TrimRight(principal.ActorURL, "/") + "/activities/" + kind + "/" + token

		tag, err := deps.PG.Exec(r.Context(), `
INSERT INTO reactions (actor_id, note_id, kind, activity_url)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
`,
			principal.ActorID,
			target.ID,
			kind,
			activityID,
		)
		if err != nil {
			deps.Logger.Error("insert reaction failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if tag.RowsAffected() > 0 {
			if target.ActorLocal && target.ActorID != principal.ActorID {
				actorID := principal.ActorID
				noteIDCopy := target.ID
				if err := notifications.Insert(r.Context(), deps.PG, target.ActorID, kind, &actorID, &noteIDCopy); err != nil {
					deps.Logger.Warn("insert reaction notification failed", "error", err)
				}
			}

			if !target.ActorLocal {
				targetInbox := strings.TrimSpace(target.ActorInbox)
				if targetInbox == "" {
					targetInbox = strings.TrimRight(target.ActorURL, "/") + "/inbox"
				}
				if targetInbox != "" {
					payload := map[string]any{
						"@context": "https://www.w3.org/ns/activitystreams",
						"id":       activityID,
						"type":     activityType,
						"actor":    principal.ActorURL,
						"object":   target.NoteURL,
					}
					if raw, err := json.Marshal(payload); err == nil {
						if _, err := deps.PG.Exec(r.Context(), `
INSERT INTO deliveries (target_inbox, activity_id, activity_json)
VALUES ($1, $2, $3)
`,
							targetInbox,
							activityID,
							raw,
						); err != nil {
							deps.Logger.Warn("enqueue reaction delivery failed", "error", err)
						}
					}
				}
			}
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"note_id":     target.ID,
			"kind":        kind,
			"activity_id": activityID,
		})
	}
}

func handleDeleteReaction(deps Dependencies, kind string) http.HandlerFunc {
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
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		var activityURL string
		err = deps.PG.QueryRow(r.Context(), `
SELECT COALESCE(activity_url, '')
FROM reactions
WHERE actor_id = $1
  AND note_id = $2
  AND kind = $3
`,
			principal.ActorID,
			target.ID,
			kind,
		).Scan(&activityURL)
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := deps.PG.Exec(r.Context(), `
DELETE FROM reactions
WHERE actor_id = $1
  AND note_id = $2
  AND kind = $3
`,
			principal.ActorID,
			target.ID,
			kind,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if !target.ActorLocal && activityURL != "" {
			targetInbox := strings.TrimSpace(target.ActorInbox)
			if targetInbox == "" {
				targetInbox = strings.TrimRight(target.ActorURL, "/") + "/inbox"
			}
			if targetInbox != "" {
				token, err := randomToken(12)
				if err == nil {
					undoID := strings.TrimRight(principal.ActorURL, "/") + "/activities/undo/" + token
					undo := map[string]any{
						"@context": "https://www.w3.org/ns/activitystreams",
						"id":       undoID,
						"type":     "Undo",
						"actor":    principal.ActorURL,
						"object": map[string]any{
							"id":   activityURL,
							"type": strings.Title(kind),
						},
					}
					if raw, err := json.Marshal(undo); err == nil {
						if _, err := deps.PG.Exec(r.Context(), `
INSERT INTO deliveries (target_inbox, activity_id, activity_json)
VALUES ($1, $2, $3)
`,
							targetInbox,
							undoID,
							raw,
						); err != nil {
							deps.Logger.Warn("enqueue reaction undo failed", "error", err)
						}
					}
				}
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleDeleteOwnPost(deps Dependencies) http.HandlerFunc {
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

		var noteURL string
		err = deps.PG.QueryRow(r.Context(), `
SELECT COALESCE(note_url, '')
FROM notes
WHERE id = $1
  AND actor_id = $2
`,
			noteID,
			principal.ActorID,
		).Scan(&noteURL)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "note_not_found"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		if noteURL == "" {
			noteURL = strings.TrimRight(deps.Config.AppBaseURL, "/") + "/notes/" + strconv.FormatInt(noteID, 10)
		}

		if _, err := deps.PG.Exec(r.Context(), `
DELETE FROM notes
WHERE id = $1
  AND actor_id = $2
`,
			noteID,
			principal.ActorID,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		var followers []actorTarget
		rows, err := deps.PG.Query(r.Context(), `
SELECT a.id, a.local, COALESCE(a.username, ''), COALESCE(a.actor_url, ''), COALESCE(a.inbox_url, '')
FROM follows f
JOIN actors a ON a.id = f.follower_id
WHERE f.following_id = $1
  AND f.state = 'accepted'
`,
			principal.ActorID,
		)
		if err == nil {
			for rows.Next() {
				var f actorTarget
				if scanErr := rows.Scan(&f.ID, &f.Local, &f.Username, &f.ActorURL, &f.InboxURL); scanErr == nil {
					followers = append(followers, f)
				}
			}
			rows.Close()
		}

		token, err := randomToken(12)
		if err == nil {
			deleteActivityID := strings.TrimRight(principal.ActorURL, "/") + "/activities/delete/" + token
			deleteActivity := map[string]any{
				"@context": "https://www.w3.org/ns/activitystreams",
				"id":       deleteActivityID,
				"type":     "Delete",
				"actor":    principal.ActorURL,
				"object":   noteURL,
			}
			if raw, err := json.Marshal(deleteActivity); err == nil {
				for _, follower := range followers {
					if follower.Local {
						continue
					}
					targetInbox := strings.TrimSpace(follower.InboxURL)
					if targetInbox == "" && follower.ActorURL != "" {
						targetInbox = strings.TrimRight(follower.ActorURL, "/") + "/inbox"
					}
					if targetInbox == "" {
						continue
					}
					_, _ = deps.PG.Exec(r.Context(), `
INSERT INTO deliveries (target_inbox, activity_id, activity_json)
VALUES ($1, $2, $3)
`,
						targetInbox,
						deleteActivityID,
						raw,
					)
				}
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func resolveFollowTarget(ctx context.Context, deps Dependencies, rawTarget string, allowFetch bool) (actorTarget, error) {
	target := strings.TrimSpace(rawTarget)
	if target == "" {
		return actorTarget{}, errors.New("target_required")
	}

	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		parsed, err := url.Parse(target)
		if err != nil || parsed.Host == "" {
			return actorTarget{}, errors.New("invalid_target")
		}
		if !strings.EqualFold(parsed.Scheme, "https") {
			return actorTarget{}, errors.New("remote_target_must_use_https")
		}

		var existing actorTarget
		err = deps.PG.QueryRow(ctx, `
SELECT id, local, COALESCE(username, ''), COALESCE(actor_url, ''), COALESCE(inbox_url, '')
FROM actors
WHERE actor_url = $1
LIMIT 1
`,
			target,
		).Scan(&existing.ID, &existing.Local, &existing.Username, &existing.ActorURL, &existing.InboxURL)
		if err == nil {
			return existing, nil
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return actorTarget{}, errors.New("target_lookup_failed")
		}
		if !allowFetch {
			return actorTarget{}, errors.New("target_not_found")
		}

		doc, err := fetch.DerefActor(ctx, target)
		if err != nil {
			return actorTarget{}, errors.New("remote_actor_fetch_failed")
		}
		targetActorURL := strings.TrimSpace(doc.ID)
		if targetActorURL == "" {
			targetActorURL = target
		}

		var resolved actorTarget
		err = deps.PG.QueryRow(ctx, `
INSERT INTO actors (
  local,
  username,
  domain,
  display_name,
  summary,
  actor_url,
  inbox_url
) VALUES (
  FALSE,
  NULL,
  $1,
  '',
  '',
  $2,
  NULLIF($3, '')
)
ON CONFLICT (actor_url)
DO UPDATE SET
  inbox_url = CASE
    WHEN COALESCE(actors.inbox_url, '') = '' THEN NULLIF(EXCLUDED.inbox_url, '')
    ELSE actors.inbox_url
  END,
  updated_at = now()
RETURNING id, local, COALESCE(username, ''), COALESCE(actor_url, ''), COALESCE(inbox_url, '')
`,
			parsed.Host,
			targetActorURL,
			strings.TrimSpace(doc.Inbox),
		).Scan(&resolved.ID, &resolved.Local, &resolved.Username, &resolved.ActorURL, &resolved.InboxURL)
		if err != nil {
			return actorTarget{}, errors.New("target_upsert_failed")
		}
		return resolved, nil
	}

	var local actorTarget
	err := deps.PG.QueryRow(ctx, `
SELECT id, local, COALESCE(username, ''), COALESCE(actor_url, ''), COALESCE(inbox_url, '')
FROM actors
WHERE local = TRUE
  AND domain = $1
  AND username = $2
LIMIT 1
`,
		deps.Config.AppDomain,
		strings.ToLower(target),
	).Scan(&local.ID, &local.Local, &local.Username, &local.ActorURL, &local.InboxURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return actorTarget{}, errors.New("target_not_found")
	}
	if err != nil {
		return actorTarget{}, errors.New("target_lookup_failed")
	}
	return local, nil
}

func noteIDFromPath(r *http.Request) (int64, bool) {
	idRaw := strings.TrimSpace(r.PathValue("id"))
	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func loadNoteTarget(ctx context.Context, deps Dependencies, noteID int64) (noteTarget, error) {
	var n noteTarget
	err := deps.PG.QueryRow(ctx, `
SELECT
  n.id,
  COALESCE(n.note_url, ''),
  a.id,
  a.local,
  COALESCE(a.actor_url, ''),
  COALESCE(a.inbox_url, ''),
  n.published_at
FROM notes n
JOIN actors a ON a.id = n.actor_id
WHERE n.id = $1
`,
		noteID,
	).Scan(
		&n.ID,
		&n.NoteURL,
		&n.ActorID,
		&n.ActorLocal,
		&n.ActorURL,
		&n.ActorInbox,
		&n.PublishedAt,
	)
	if err != nil {
		return noteTarget{}, err
	}
	if n.NoteURL == "" {
		n.NoteURL = strings.TrimRight(deps.Config.AppBaseURL, "/") + "/notes/" + strconv.FormatInt(n.ID, 10)
	}
	return n, nil
}
