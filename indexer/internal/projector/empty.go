package projector


import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// Empty empties every table the projector writes, and its cursors, so the
// next run reads every node's log from the start. Only the record of the
// migrations is kept. Every table is a projection of the logs: what Empty
// drops, reading the logs brings back.
func Empty(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `
		SELECT tablename FROM pg_tables
		WHERE schemaname = current_schema()
		  AND tablename <> 'goose_db_version'`)
	if err != nil {
		return err
	}

	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}

	if len(tables) == 0 {
		return nil
	}

	quoted := make([]string, len(tables))
	for i, t := range tables {
		quoted[i] = pgx.Identifier{t}.Sanitize()
	}

	_, err = pool.Exec(ctx, "TRUNCATE "+strings.Join(quoted, ", "))
	if err != nil {
		return fmt.Errorf("empty the projection: %w", err)
	}

	return nil
}
