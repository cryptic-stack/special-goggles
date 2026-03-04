package handlers

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func noteAttachments(ctx context.Context, pool *pgxpool.Pool, baseURL string, noteID int64) ([]map[string]any, error) {
	rows, err := pool.Query(ctx, `
SELECT m.storage_key, m.content_type, COALESCE(m.original_name, '')
FROM note_attachments na
JOIN media_attachments m ON m.id = na.attachment_id
WHERE na.note_id = $1
ORDER BY m.id ASC
`,
		noteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]any, 0, 2)
	for rows.Next() {
		var (
			storageKey  string
			contentType string
			original    string
		)
		if err := rows.Scan(&storageKey, &contentType, &original); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"type":      "Document",
			"mediaType": contentType,
			"url":       mediaURL(baseURL, storageKey),
			"name":      original,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func mediaURL(baseURL, storageKey string) string {
	return strings.TrimRight(baseURL, "/") + "/media/" + strings.TrimLeft(strings.ReplaceAll(storageKey, "\\", "/"), "/")
}
