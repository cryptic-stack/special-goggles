package accounts

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func SeedDevAlice(ctx context.Context, pool *pgxpool.Pool, appBaseURL, appDomain, devPassword string, logger *slog.Logger) error {
	var localCount int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM actors WHERE local = TRUE`).Scan(&localCount); err != nil {
		return fmt.Errorf("count local actors: %w", err)
	}

	var actorID int64
	err := pool.QueryRow(ctx, `
SELECT id
FROM actors
WHERE local = TRUE
  AND username = 'alice'
  AND domain = $1
LIMIT 1
`,
		appDomain,
	).Scan(&actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		if localCount > 0 {
			return nil
		}

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return fmt.Errorf("generate rsa keypair: %w", err)
		}

		publicKeyPEM, err := encodePublicKeyPEM(&privateKey.PublicKey)
		if err != nil {
			return err
		}
		privateKeyPEM := encodePrivateKeyPEM(privateKey)

		base := strings.TrimRight(appBaseURL, "/")
		actorURL := base + "/users/alice"

		err = pool.QueryRow(ctx, `
INSERT INTO actors (
  local,
  username,
  domain,
  display_name,
  summary,
  actor_url,
  inbox_url,
  outbox_url,
  followers_url,
  following_url,
  public_key_pem,
  private_key_pem
) VALUES (
  TRUE,
  'alice',
  $1,
  'Alice',
  '',
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8
)
RETURNING id
`,
			appDomain,
			actorURL,
			actorURL+"/inbox",
			actorURL+"/outbox",
			actorURL+"/followers",
			actorURL+"/following",
			publicKeyPEM,
			privateKeyPEM,
		).Scan(&actorID)
		if err != nil {
			return fmt.Errorf("insert alice actor: %w", err)
		}

		logger.Info("seeded dev actor", "username", "alice", "actor_url", actorURL)
	} else if err != nil {
		return fmt.Errorf("lookup alice actor: %w", err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(devPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash dev password: %w", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO users (actor_id, email, password_hash)
VALUES ($1, $2, $3)
ON CONFLICT (actor_id)
DO UPDATE SET
  email = EXCLUDED.email,
  password_hash = EXCLUDED.password_hash
`,
		actorID,
		"alice@"+appDomain,
		string(passwordHash),
	); err != nil {
		return fmt.Errorf("insert alice user: %w", err)
	}

	return nil
}

func encodePublicKeyPEM(publicKey *rsa.PublicKey) (string, error) {
	derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("marshal public key: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	})), nil
}

func encodePrivateKeyPEM(privateKey *rsa.PrivateKey) string {
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}))
}
