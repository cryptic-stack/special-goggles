package notifications

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Insert(ctx context.Context, pool *pgxpool.Pool, userActorID int64, kind string, actorID *int64, noteID *int64) error {
	_, err := pool.Exec(ctx, `
INSERT INTO notifications (user_actor_id, kind, actor_id, note_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
`,
		userActorID,
		kind,
		actorID,
		noteID,
	)
	return err
}
