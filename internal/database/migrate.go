package database

import (
	"apiproject/migrations"
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

func Migrate(ctx context.Context, db *sqlx.DB) error {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("create migration locker: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations.FS,
		goose.WithSessionLocker(locker),
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	for _, r := range results {
		slog.InfoContext(ctx, "migration applied",
			"version", r.Source.Version,
			"file", r.Source.Path,
			"duration", r.Duration,
		)
	}

	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("get schema version: %w", err)
	}
	slog.InfoContext(ctx, "database schema is up to date", "version", version)
	return nil
}
