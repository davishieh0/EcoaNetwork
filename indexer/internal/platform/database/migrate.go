// Package database is the indexer's Postgres: the migrations, and later the
// pool the subjects read and write through.
package database


import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)


// --- CODE ---

// Migrate applies every migration the database at url does not have yet.
//
// goose works on database/sql, so this goes through pgx's database/sql
// driver; the rest of the indexer uses pgx directly.
func Migrate(ctx context.Context, url string) error {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres,
	                                   db,
	                                   migrations.FS)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	for _, r := range results {
		slog.Info("migrated",
		          "version", r.Source.Version,
		          "file", r.Source.Path,
		          "took", r.Duration.String())
	}

	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("read version: %w", err)
	}

	slog.Info("database up to date",
	          "version", version,
	          "applied", len(results))

	return nil
}
