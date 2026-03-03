package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/cryptic-stack/special-goggles/backend/internal/ap/fetch"
	"github.com/cryptic-stack/special-goggles/backend/internal/ap/signatures"
	"github.com/cryptic-stack/special-goggles/backend/internal/domain/notifications"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

type inboxActivity struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Actor  json.RawMessage `json:"actor"`
	Object json.RawMessage `json:"object"`
}

type inboxObjectRef struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type inboxActorRef struct {
	ID    string `json:"id"`
	Inbox string `json:"inbox"`
}

type inboxNoteObject struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	AttributedTo string `json:"attributedTo"`
	InReplyTo    string `json:"inReplyTo"`
	Content      string `json:"content"`
	Published    string `json:"published"`
	Sensitive    bool   `json:"sensitive"`
}

type localInboxActor struct {
	ID       int64
	ActorURL string
}

func Inbox(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := strings.TrimSpace(r.PathValue("username"))
		if username == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid_username",
			})
			return
		}

		localActor, err := lookupLocalActor(r.Context(), deps.PG, username, deps.Config.AppDomain)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "actor_not_found",
			})
			return
		}
		if err != nil {
			deps.Logger.Error("inbox local actor lookup failed", "error", err, "username", username)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal_server_error",
			})
			return
		}

		body, err := readInboxBody(w, r, deps.Config.InboxMaxBody)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{
					"error": "body_too_large",
				})
				return
			}

			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid_json",
			})
			return
		}

		if err := verifyInboundHTTPSignature(r.Context(), deps, r, body); err != nil {
			if deps.Config.APAllowUnsignedInbound && errors.Is(err, signatures.ErrMissingSignature) {
				deps.Logger.Warn("accepting unsigned inbox request due dev flag",
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
				)
			} else {
				deps.Logger.Warn("inbox signature verification failed",
					"error", err,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
				)
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "invalid_signature",
				})
				return
			}
		}

		var activity inboxActivity
		if err := json.Unmarshal(body, &activity); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid_json",
			})
			return
		}
		activity.ID = strings.TrimSpace(activity.ID)
		activity.Type = strings.TrimSpace(activity.Type)
		if activity.ID == "" || activity.Type == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "missing_id_or_type",
			})
			return
		}

		actorURL, _, _ := parseActorRef(activity.Actor)
		inserted, err := persistInboxActivity(r.Context(), deps.PG, activity.ID, actorURL, activity.Type, body)
		if err != nil {
			deps.Logger.Error("failed to persist inbox activity", "error", err, "activity_id", activity.ID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal_server_error",
			})
			return
		}
		if !inserted {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		if err := handleInboxActivity(r.Context(), deps, localActor, activity); err != nil {
			deps.Logger.Error("inbox activity handling failed",
				"error", err,
				"activity_id", activity.ID,
				"type", activity.Type,
			)
		}

		w.WriteHeader(http.StatusAccepted)
	})
}

func handleInboxActivity(ctx context.Context, deps Dependencies, localActor localInboxActor, activity inboxActivity) error {
	switch strings.ToLower(activity.Type) {
	case "follow":
		return handleFollow(ctx, deps, localActor, activity)
	case "undo":
		return handleUndo(ctx, deps, localActor, activity)
	case "accept":
		return handleAccept(ctx, deps, activity)
	case "like", "announce":
		return handleReaction(ctx, deps, activity)
	case "create":
		return handleCreate(ctx, deps, activity)
	case "delete":
		return handleDelete(ctx, deps, activity)
	default:
		return nil
	}
}

func handleFollow(ctx context.Context, deps Dependencies, localActor localInboxActor, activity inboxActivity) error {
	remoteActorURL, inboxHint, err := parseActorRef(activity.Actor)
	if err != nil || remoteActorURL == "" {
		return nil
	}

	targetActorURL, _, err := parseObjectRef(activity.Object)
	if err != nil || targetActorURL == "" {
		return nil
	}
	if targetActorURL != localActor.ActorURL {
		return nil
	}

	remoteActorID, remoteInbox, err := ensureRemoteActor(ctx, deps.PG, remoteActorURL, inboxHint)
	if err != nil {
		return err
	}
	if remoteActorID == 0 {
		return nil
	}

	if _, err := deps.PG.Exec(ctx, `
INSERT INTO follows (follower_id, following_id, state, follow_activity_url)
VALUES ($1, $2, 'accepted', $3)
ON CONFLICT (follower_id, following_id)
DO UPDATE SET
  state = 'accepted',
  follow_activity_url = EXCLUDED.follow_activity_url
`,
		remoteActorID,
		localActor.ID,
		activity.ID,
	); err != nil {
		return fmt.Errorf("upsert follow relation: %w", err)
	}

	if err := notifications.Insert(ctx, deps.PG, localActor.ID, "follow", &remoteActorID, nil); err != nil {
		deps.Logger.Warn("insert follow notification failed", "error", err)
	}

	acceptActivityID, err := newActivityID(localActor.ActorURL, "accept")
	if err != nil {
		return err
	}

	accept := map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       acceptActivityID,
		"type":     "Accept",
		"actor":    localActor.ActorURL,
		"object": map[string]any{
			"id":     activity.ID,
			"type":   "Follow",
			"actor":  remoteActorURL,
			"object": targetActorURL,
		},
	}

	targetInbox := remoteInbox
	if targetInbox == "" {
		targetInbox = fallbackInboxURL(remoteActorURL)
	}
	if targetInbox == "" {
		deps.Logger.Warn("cannot enqueue accept; remote inbox unknown", "remote_actor", remoteActorURL)
		return nil
	}

	return enqueueDelivery(ctx, deps.PG, targetInbox, acceptActivityID, accept)
}

func handleAccept(ctx context.Context, deps Dependencies, activity inboxActivity) error {
	remoteActorURL, _, err := parseActorRef(activity.Actor)
	if err != nil || remoteActorURL == "" {
		return nil
	}

	remoteActorID, _, err := ensureRemoteActor(ctx, deps.PG, remoteActorURL, "")
	if err != nil {
		return err
	}
	if remoteActorID == 0 {
		return nil
	}

	objectID, objectType, err := parseObjectRef(activity.Object)
	if err != nil || objectID == "" {
		return nil
	}
	if objectType != "" && !strings.EqualFold(objectType, "follow") {
		return nil
	}

	_, err = deps.PG.Exec(ctx, `
UPDATE follows
SET state = 'accepted'
WHERE following_id = $1
  AND follow_activity_url = $2
`,
		remoteActorID,
		objectID,
	)
	return err
}

func handleUndo(ctx context.Context, deps Dependencies, _ localInboxActor, activity inboxActivity) error {
	remoteActorURL, _, _ := parseActorRef(activity.Actor)
	if remoteActorURL == "" {
		return nil
	}

	remoteActorID, _, err := ensureRemoteActor(ctx, deps.PG, remoteActorURL, "")
	if err != nil {
		return err
	}
	if remoteActorID == 0 {
		return nil
	}

	objectID, objectType, err := parseObjectRef(activity.Object)
	if err != nil || objectID == "" {
		return nil
	}

	switch strings.ToLower(objectType) {
	case "follow":
		_, err = deps.PG.Exec(ctx,
			`DELETE FROM follows WHERE follower_id = $1 AND follow_activity_url = $2`,
			remoteActorID, objectID,
		)
		return err
	case "like", "announce":
		_, err = deps.PG.Exec(ctx,
			`DELETE FROM reactions WHERE actor_id = $1 AND activity_url = $2`,
			remoteActorID, objectID,
		)
		return err
	default:
		if _, err := deps.PG.Exec(ctx,
			`DELETE FROM follows WHERE follower_id = $1 AND follow_activity_url = $2`,
			remoteActorID, objectID,
		); err != nil {
			return err
		}
		_, err = deps.PG.Exec(ctx,
			`DELETE FROM reactions WHERE actor_id = $1 AND activity_url = $2`,
			remoteActorID, objectID,
		)
		return err
	}
}

func handleReaction(ctx context.Context, deps Dependencies, activity inboxActivity) error {
	remoteActorURL, _, err := parseActorRef(activity.Actor)
	if err != nil || remoteActorURL == "" {
		return nil
	}

	remoteActorID, _, err := ensureRemoteActor(ctx, deps.PG, remoteActorURL, "")
	if err != nil {
		return err
	}
	if remoteActorID == 0 {
		return nil
	}

	noteURL, _, err := parseObjectRef(activity.Object)
	if err != nil || noteURL == "" {
		return nil
	}

	var noteID int64
	if err := deps.PG.QueryRow(ctx, `SELECT id FROM notes WHERE note_url = $1`, noteURL).Scan(&noteID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	kind := strings.ToLower(activity.Type)
	_, err = deps.PG.Exec(ctx, `
INSERT INTO reactions (actor_id, note_id, kind, activity_url)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
`,
		remoteActorID,
		noteID,
		kind,
		activity.ID,
	)
	if err != nil {
		return err
	}

	var noteAuthorID int64
	if err := deps.PG.QueryRow(ctx, `SELECT actor_id FROM notes WHERE id = $1`, noteID).Scan(&noteAuthorID); err == nil {
		if noteAuthorID != remoteActorID {
			_ = notifications.Insert(ctx, deps.PG, noteAuthorID, kind, &remoteActorID, &noteID)
		}
	}

	return err
}

func handleCreate(ctx context.Context, deps Dependencies, activity inboxActivity) error {
	noteObject, ok := parseNoteObject(activity.Object)
	if !ok {
		return nil
	}

	if noteObject.ID == "" {
		return nil
	}
	if noteObject.Type != "" && !strings.EqualFold(noteObject.Type, "Note") {
		return nil
	}

	remoteActorURL := strings.TrimSpace(noteObject.AttributedTo)
	actorURLFromActivity, inboxHint, _ := parseActorRef(activity.Actor)
	if remoteActorURL == "" {
		remoteActorURL = actorURLFromActivity
	}
	if remoteActorURL == "" {
		return nil
	}

	remoteActorID, _, err := ensureRemoteActor(ctx, deps.PG, remoteActorURL, inboxHint)
	if err != nil {
		return err
	}
	if remoteActorID == 0 {
		return nil
	}

	publishedAt := time.Now().UTC()
	if noteObject.Published != "" {
		parsed, err := time.Parse(time.RFC3339, noteObject.Published)
		if err == nil {
			publishedAt = parsed.UTC()
		}
	}

	contentHTML := noteObject.Content
	contentText := strings.TrimSpace(htmlTagPattern.ReplaceAllString(noteObject.Content, ""))
	if contentText == "" {
		contentText = contentHTML
	}

	var insertedNoteID int64
	err = deps.PG.QueryRow(ctx, `
INSERT INTO notes (
  local,
  note_url,
  actor_id,
  in_reply_to_url,
  content_html,
  content_text,
  visibility,
  sensitive,
  published_at
) VALUES (
  FALSE,
  $1,
  $2,
  NULLIF($3, ''),
  $4,
  $5,
  'public',
  $6,
  $7
)
ON CONFLICT (note_url) DO NOTHING
RETURNING id
`,
		noteObject.ID,
		remoteActorID,
		noteObject.InReplyTo,
		contentHTML,
		contentText,
		noteObject.Sensitive,
		publishedAt,
	).Scan(&insertedNoteID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if insertedNoteID == 0 {
		if err := deps.PG.QueryRow(ctx, `SELECT id FROM notes WHERE note_url = $1`, noteObject.ID).Scan(&insertedNoteID); err != nil {
			return nil
		}
	}

	_, _ = deps.PG.Exec(ctx, `
INSERT INTO timeline_items (user_actor_id, note_id)
SELECT f.follower_id, $2
FROM follows f
JOIN actors a ON a.id = f.follower_id
WHERE f.following_id = $1
  AND f.state = 'accepted'
  AND a.local = TRUE
ON CONFLICT DO NOTHING
`,
		remoteActorID,
		insertedNoteID,
	)

	if noteObject.InReplyTo != "" {
		var parentActorID int64
		if err := deps.PG.QueryRow(ctx, `SELECT actor_id FROM notes WHERE note_url = $1`, noteObject.InReplyTo).Scan(&parentActorID); err == nil {
			if parentActorID != remoteActorID {
				_ = notifications.Insert(ctx, deps.PG, parentActorID, "reply", &remoteActorID, &insertedNoteID)
			}
		}
	}

	return nil
}

func handleDelete(ctx context.Context, deps Dependencies, activity inboxActivity) error {
	objectID, objectType, err := parseObjectRef(activity.Object)
	if err != nil || objectID == "" {
		return nil
	}

	switch strings.ToLower(objectType) {
	case "actor", "person", "group", "service", "application":
		_, err = deps.PG.Exec(ctx, `DELETE FROM actors WHERE actor_url = $1 AND local = FALSE`, objectID)
		return err
	default:
		_, err = deps.PG.Exec(ctx, `DELETE FROM notes WHERE note_url = $1`, objectID)
		return err
	}
}

func readInboxBody(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, error) {
	if r.Body == nil {
		return nil, errors.New("missing request body")
	}

	limitedBody := http.MaxBytesReader(w, r.Body, maxBytes)
	defer limitedBody.Close()

	body, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, errors.New("empty request body")
	}
	return body, nil
}

func persistInboxActivity(ctx context.Context, pool *pgxpool.Pool, activityID, actorURL, activityType string, raw []byte) (bool, error) {
	tag, err := pool.Exec(ctx, `
INSERT INTO inbox_activities (activity_id, actor_url, type, raw_json)
VALUES ($1, $2, $3, $4)
ON CONFLICT (activity_id) DO NOTHING
`,
		activityID,
		emptyToNull(actorURL),
		activityType,
		raw,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func lookupLocalActor(ctx context.Context, pool *pgxpool.Pool, username, domain string) (localInboxActor, error) {
	var actor localInboxActor
	err := pool.QueryRow(ctx, `
SELECT id, actor_url
FROM actors
WHERE local = TRUE
  AND username = $1
  AND domain = $2
`,
		username, domain,
	).Scan(&actor.ID, &actor.ActorURL)
	if err != nil {
		return localInboxActor{}, err
	}
	return actor, nil
}

func ensureRemoteActor(ctx context.Context, pool *pgxpool.Pool, actorURL, inboxHint string) (int64, string, error) {
	parsed, err := url.Parse(actorURL)
	if err != nil || parsed.Host == "" {
		return 0, "", nil
	}

	var id int64
	var inbox string
	err = pool.QueryRow(ctx, `
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
  inbox_url = COALESCE(actors.inbox_url, EXCLUDED.inbox_url),
  updated_at = now()
RETURNING id, COALESCE(inbox_url, '')
`,
		parsed.Host,
		actorURL,
		inboxHint,
	).Scan(&id, &inbox)
	if err != nil {
		return 0, "", err
	}
	return id, inbox, nil
}

func parseActorRef(raw json.RawMessage) (id string, inbox string, err error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", "", errors.New("actor missing")
	}

	switch raw[0] {
	case '"':
		var actorURL string
		if err := json.Unmarshal(raw, &actorURL); err != nil {
			return "", "", err
		}
		return strings.TrimSpace(actorURL), "", nil
	case '{':
		var actor inboxActorRef
		if err := json.Unmarshal(raw, &actor); err != nil {
			return "", "", err
		}
		return strings.TrimSpace(actor.ID), strings.TrimSpace(actor.Inbox), nil
	default:
		return "", "", errors.New("invalid actor format")
	}
}

func parseObjectRef(raw json.RawMessage) (id string, activityType string, err error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", "", errors.New("object missing")
	}

	switch raw[0] {
	case '"':
		var objectID string
		if err := json.Unmarshal(raw, &objectID); err != nil {
			return "", "", err
		}
		return strings.TrimSpace(objectID), "", nil
	case '{':
		var object inboxObjectRef
		if err := json.Unmarshal(raw, &object); err != nil {
			return "", "", err
		}
		return strings.TrimSpace(object.ID), strings.TrimSpace(object.Type), nil
	default:
		return "", "", errors.New("invalid object format")
	}
}

func parseNoteObject(raw json.RawMessage) (inboxNoteObject, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return inboxNoteObject{}, false
	}
	if raw[0] != '{' {
		return inboxNoteObject{}, false
	}

	var note inboxNoteObject
	if err := json.Unmarshal(raw, &note); err != nil {
		return inboxNoteObject{}, false
	}
	return note, true
}

func enqueueDelivery(ctx context.Context, pool *pgxpool.Pool, targetInbox, activityID string, payload any) error {
	if targetInbox == "" || activityID == "" {
		return nil
	}

	activityJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
INSERT INTO deliveries (target_inbox, activity_id, activity_json)
VALUES ($1, $2, $3)
`,
		targetInbox,
		activityID,
		activityJSON,
	)
	return err
}

func newActivityID(baseActorURL, kind string) (string, error) {
	token, err := randomHex(16)
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(baseActorURL, "/")
	return base + "/activities/" + kind + "/" + token, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func fallbackInboxURL(actorURL string) string {
	parsed, err := url.Parse(actorURL)
	if err != nil {
		return ""
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return strings.TrimRight(actorURL, "/") + "/inbox"
}

func emptyToNull(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func verifyInboundHTTPSignature(ctx context.Context, deps Dependencies, req *http.Request, body []byte) error {
	return signatures.VerifyRequest(
		ctx,
		req,
		body,
		time.Duration(deps.Config.APSignatureMaxSkewSec)*time.Second,
		func(ctx context.Context, keyID string) (string, error) {
			return resolvePublicKeyByKeyID(ctx, deps, keyID)
		},
	)
}

func resolvePublicKeyByKeyID(ctx context.Context, deps Dependencies, keyID string) (string, error) {
	ownerURL := keyOwnerURL(keyID)
	if ownerURL == "" {
		return "", signatures.ErrUnknownKey
	}

	publicKeyPEM, err := lookupPublicKeyByActorURL(ctx, deps.PG, ownerURL)
	if err != nil {
		return "", err
	}
	if publicKeyPEM != "" {
		return publicKeyPEM, nil
	}

	actorDoc, err := fetch.DerefActor(ctx, ownerURL)
	if err != nil {
		return "", err
	}

	keyPEM := strings.TrimSpace(actorDoc.PublicKey.PublicKeyPEM)
	if keyPEM == "" {
		return "", signatures.ErrUnknownKey
	}

	if actorDoc.PublicKey.ID != "" && keyID != "" && actorDoc.PublicKey.ID != keyID {
		return "", fmt.Errorf("fetched actor key id mismatch")
	}

	if err := upsertFetchedRemoteActor(ctx, deps.PG, actorDoc); err != nil {
		return "", err
	}

	return keyPEM, nil
}

func lookupPublicKeyByActorURL(ctx context.Context, pool *pgxpool.Pool, actorURL string) (string, error) {
	var key string
	err := pool.QueryRow(ctx, `
SELECT COALESCE(public_key_pem, '')
FROM actors
WHERE actor_url = $1
LIMIT 1
`,
		actorURL,
	).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(key), nil
}

func upsertFetchedRemoteActor(ctx context.Context, pool *pgxpool.Pool, actor fetch.ActorDocument) error {
	actorID := strings.TrimSpace(actor.ID)
	if actorID == "" {
		return fmt.Errorf("fetched actor missing id")
	}

	parsed, err := url.Parse(actorID)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid fetched actor id")
	}

	_, err = pool.Exec(ctx, `
INSERT INTO actors (
  local,
  username,
  domain,
  display_name,
  summary,
  actor_url,
  inbox_url,
  public_key_pem
) VALUES (
  FALSE,
  NULL,
  $1,
  '',
  '',
  $2,
  NULLIF($3, ''),
  $4
)
ON CONFLICT (actor_url)
DO UPDATE SET
  inbox_url = CASE
    WHEN COALESCE(actors.inbox_url, '') = '' THEN NULLIF(EXCLUDED.inbox_url, '')
    ELSE actors.inbox_url
  END,
  public_key_pem = EXCLUDED.public_key_pem,
  updated_at = now()
`,
		parsed.Host,
		actorID,
		actor.Inbox,
		actor.PublicKey.PublicKeyPEM,
	)
	return err
}

func keyOwnerURL(keyID string) string {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return ""
	}

	parsed, err := url.Parse(keyID)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	parsed.Fragment = ""
	return parsed.String()
}
