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

-- name: ObserveEvent :exec
-- Records when a node had the event, keeping the earliest any node reported.
INSERT INTO event_observations (event_id, received_at)
VALUES ($1, $2)
ON CONFLICT (event_id) DO UPDATE
SET received_at = LEAST(event_observations.received_at, EXCLUDED.received_at);

-- name: HoldEvent :exec
-- Records that a node holds the event, where in its log and when it had it.
INSERT INTO event_holders (event_id, node, sequence, received_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (event_id, node) DO UPDATE
SET sequence = EXCLUDED.sequence, received_at = EXCLUDED.received_at;

-- name: HoldersOf :many
-- The nodes that hold an event, the first to have it first.
SELECT node, sequence, received_at
FROM event_holders
WHERE event_id = $1
ORDER BY received_at NULLS LAST, node;
