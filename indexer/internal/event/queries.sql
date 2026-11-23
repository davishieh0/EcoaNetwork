-- The queries of the events table. `make sqlc` turns each into a function of
-- the generated package, which repository.go wraps.

-- name: EventByID :one
-- One event, with when the network had it if any node reported it.
SELECT e.event_id, e.envelope, o.received_at
FROM events e
LEFT JOIN event_observations o USING (event_id)
WHERE e.event_id = $1;

-- name: SaveEvent :execrows
-- Stores an event once: a copy from another node, or the same one read again,
-- changes nothing. The count of rows tells the two apart.
INSERT INTO events (event_id, event_type, author, created_at, envelope)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (event_id) DO NOTHING;
