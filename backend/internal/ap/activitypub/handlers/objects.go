package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func NoteObject(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idParam := strings.TrimSpace(r.PathValue("id"))
		noteID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil || noteID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_note_id"})
			return
		}

		var (
			noteURL      string
			contentHTML  string
			inReplyTo    string
			sensitive    bool
			publishedAt  time.Time
			attributedTo string
			visibility   string
		)

		err = deps.PG.QueryRow(r.Context(), `
SELECT
  COALESCE(n.note_url, ''),
  n.content_html,
  COALESCE(n.in_reply_to_url, ''),
  n.sensitive,
  n.published_at,
  COALESCE(a.actor_url, ''),
  COALESCE(n.visibility, 'public')
FROM notes n
JOIN actors a ON a.id = n.actor_id
WHERE n.id = $1
`,
			noteID,
		).Scan(
			&noteURL,
			&contentHTML,
			&inReplyTo,
			&sensitive,
			&publishedAt,
			&attributedTo,
			&visibility,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "note_not_found"})
			return
		}
		if err != nil {
			deps.Logger.Error("note object lookup failed", "error", err, "note_id", noteID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if noteURL == "" {
			noteURL = strings.TrimRight(deps.Config.AppBaseURL, "/") + "/notes/" + strconv.FormatInt(noteID, 10)
		}

		activity := map[string]any{
			"@context":     "https://www.w3.org/ns/activitystreams",
			"id":           noteURL,
			"type":         "Note",
			"attributedTo": attributedTo,
			"content":      contentHTML,
			"published":    publishedAt.UTC().Format(time.RFC3339),
			"sensitive":    sensitive,
		}

		switch visibility {
		case "followers":
			activity["to"] = []string{}
		case "direct":
			activity["to"] = []string{}
		default:
			activity["to"] = []string{"https://www.w3.org/ns/activitystreams#Public"}
		}

		if inReplyTo != "" {
			activity["inReplyTo"] = inReplyTo
		}

		writeActivityJSON(w, http.StatusOK, activity)
	})
}
