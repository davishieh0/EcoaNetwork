-- The queries of the projector's own table. `make sqlc` turns each into a
-- function of the generated package, which repository.go wraps.

-- name: CursorOf :one
-- Where the projector stands in a node's log.
SELECT log_id, sequence FROM projector_cursors WHERE node = $1;

-- name: SaveCursor :exec
-- Moves a node's cursor, in the transaction of the event it came with.
INSERT INTO projector_cursors (node, log_id, sequence)
VALUES ($1, $2, $3)
ON CONFLICT (node) DO UPDATE
SET log_id = EXCLUDED.log_id, sequence = EXCLUDED.sequence;
