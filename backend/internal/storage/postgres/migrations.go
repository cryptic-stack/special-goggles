package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string, logger *slog.Logger) error {
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return err
	}

	files, err := readMigrationFiles(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		applied, err := isMigrationApplied(ctx, pool, file.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		body, err := os.ReadFile(file.Path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file.Name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration tx %s: %w", file.Name, err)
		}

		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", file.Name, err)
		}

		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version, filename) VALUES ($1, $2)`,
			file.Version, file.Name,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", file.Name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", file.Name, err)
		}

		logger.Info("migration applied", "version", file.Version, "file", file.Name)
	}

	return nil
}

type migrationFile struct {
	Version int64
	Name    string
	Path    string
}

func readMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	files := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		version, err := migrationVersion(entry.Name())
		if err != nil {
			return nil, err
		}

		files = append(files, migrationFile{
			Version: version,
			Name:    entry.Name(),
			Path:    filepath.Join(dir, entry.Name()),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Version < files[j].Version
	})

	return files, nil
}

func migrationVersion(name string) (int64, error) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 0 || parts[0] == "" {
		return 0, fmt.Errorf("invalid migration filename %s", name)
	}

	version, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid migration version in %s: %w", name, err)
	}
	return version, nil
}

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	const sql = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version    BIGINT PRIMARY KEY,
  filename   TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);`
	_, err := pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}
	return nil
}

func isMigrationApplied(ctx context.Context, pool *pgxpool.Pool, version int64) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check migration version %d: %w", version, err)
	}
	return exists, nil
}
