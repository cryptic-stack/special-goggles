package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cryptic-stack/special-goggles/backend/internal/ap/signatures"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultPollInterval = 5 * time.Second
	defaultBatchSize    = 25
	maxAttempts         = 12
	backoffBase         = 30 * time.Second
	backoffCap          = 6 * time.Hour
)

type Worker struct {
	pool       *pgxpool.Pool
	logger     *slog.Logger
	httpClient *http.Client
}

type deliveryRow struct {
	ID          int64
	TargetInbox string
	ActivityID  string
	ActivityRaw []byte
	Attempts    int
}

func NewWorker(pool *pgxpool.Pool, logger *slog.Logger) *Worker {
	return &Worker{
		pool:   pool,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(defaultPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processOnce(ctx); err != nil {
				w.logger.Error("delivery worker iteration failed", "error", err)
			}
		}
	}
}

func (w *Worker) processOnce(ctx context.Context) error {
	rows, err := w.pool.Query(ctx, `
SELECT id, target_inbox, activity_id, activity_json::text, attempts
FROM deliveries
WHERE state = 'queued'
  AND next_attempt_at <= now()
ORDER BY next_attempt_at ASC
LIMIT $1
`,
		defaultBatchSize,
	)
	if err != nil {
		return fmt.Errorf("query queued deliveries: %w", err)
	}
	defer rows.Close()

	var batch []deliveryRow
	for rows.Next() {
		var row deliveryRow
		if err := rows.Scan(&row.ID, &row.TargetInbox, &row.ActivityID, &row.ActivityRaw, &row.Attempts); err != nil {
			return fmt.Errorf("scan delivery row: %w", err)
		}
		batch = append(batch, row)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate delivery rows: %w", err)
	}

	for _, row := range batch {
		if err := w.processRow(ctx, row); err != nil {
			w.logger.Error("delivery processing failed",
				"delivery_id", row.ID,
				"activity_id", row.ActivityID,
				"error", err,
			)
		}
	}

	return nil
}

func (w *Worker) processRow(ctx context.Context, row deliveryRow) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, row.TargetInbox, bytes.NewReader(row.ActivityRaw))
	if err != nil {
		return w.markFailure(ctx, row, "build request: "+err.Error())
	}
	req.Header.Set("Content-Type", "application/activity+json")
	req.Header.Set("Accept", "application/activity+json, application/json")

	actorURL, err := actorURLFromActivity(row.ActivityRaw)
	if err != nil {
		return w.markFailure(ctx, row, "extract actor from activity: "+err.Error())
	}

	keyID, privateKeyPEM, err := w.lookupLocalSigningKey(ctx, actorURL)
	if err != nil {
		return w.markFailure(ctx, row, "lookup signing key: "+err.Error())
	}

	if err := signatures.SignRequest(req, row.ActivityRaw, keyID, privateKeyPEM); err != nil {
		return w.markFailure(ctx, row, "sign outbound request: "+err.Error())
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return w.markFailure(ctx, row, "send request: "+err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_, err := w.pool.Exec(ctx, `
UPDATE deliveries
SET state = 'sent',
    attempts = attempts + 1,
    last_error = '',
    next_attempt_at = now()
WHERE id = $1
`,
			row.ID,
		)
		if err != nil {
			return fmt.Errorf("mark delivery sent: %w", err)
		}
		return nil
	}

	bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	msg := fmt.Sprintf("http %d %s", resp.StatusCode, string(bodySnippet))
	return w.markFailure(ctx, row, msg)
}

func (w *Worker) markFailure(ctx context.Context, row deliveryRow, lastError string) error {
	nextAttempts := row.Attempts + 1
	if nextAttempts >= maxAttempts {
		_, err := w.pool.Exec(ctx, `
UPDATE deliveries
SET state = 'failed',
    attempts = $2,
    last_error = $3,
    next_attempt_at = now()
WHERE id = $1
`,
			row.ID,
			nextAttempts,
			lastError,
		)
		if err != nil {
			return fmt.Errorf("mark delivery failed: %w", err)
		}
		return nil
	}

	backoff := computeBackoff(nextAttempts)
	_, err := w.pool.Exec(ctx, `
UPDATE deliveries
SET state = 'queued',
    attempts = $2,
    last_error = $3,
    next_attempt_at = now() + ($4 * interval '1 second')
WHERE id = $1
`,
		row.ID,
		nextAttempts,
		lastError,
		int(backoff.Seconds()),
	)
	if err != nil {
		return fmt.Errorf("schedule delivery retry: %w", err)
	}
	return nil
}

func computeBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return backoffBase
	}

	backoff := backoffBase * time.Duration(1<<(attempt-1))
	if backoff > backoffCap || backoff < 0 {
		return backoffCap
	}
	return backoff
}

func actorURLFromActivity(raw []byte) (string, error) {
	var payload struct {
		Actor json.RawMessage `json:"actor"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", err
	}
	if len(payload.Actor) == 0 {
		return "", fmt.Errorf("missing actor field")
	}

	switch payload.Actor[0] {
	case '"':
		var actorURL string
		if err := json.Unmarshal(payload.Actor, &actorURL); err != nil {
			return "", err
		}
		actorURL = strings.TrimSpace(actorURL)
		if actorURL == "" {
			return "", fmt.Errorf("empty actor field")
		}
		return actorURL, nil
	case '{':
		var actor struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(payload.Actor, &actor); err != nil {
			return "", err
		}
		actor.ID = strings.TrimSpace(actor.ID)
		if actor.ID == "" {
			return "", fmt.Errorf("empty actor.id field")
		}
		return actor.ID, nil
	default:
		return "", fmt.Errorf("unsupported actor format")
	}
}

func (w *Worker) lookupLocalSigningKey(ctx context.Context, actorURL string) (string, string, error) {
	var privateKey string
	err := w.pool.QueryRow(ctx, `
SELECT COALESCE(private_key_pem, '')
FROM actors
WHERE local = TRUE
  AND actor_url = $1
`,
		actorURL,
	).Scan(&privateKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", fmt.Errorf("local actor not found for %s", actorURL)
	}
	if err != nil {
		return "", "", err
	}

	privateKey = strings.TrimSpace(privateKey)
	if privateKey == "" {
		return "", "", fmt.Errorf("local actor private key missing for %s", actorURL)
	}

	return strings.TrimRight(actorURL, "/") + "#main-key", privateKey, nil
}
