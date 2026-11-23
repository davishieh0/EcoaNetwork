-- The queries of the events table. `make sqlc` turns each into a function of
-- the generated package, which repository.go wraps.

-- name: EventByID :one
-- One event, with when the network had it if any node reported it.
SELECT e.event_id, e.envelope, o.received_at
FROM events e
LEFT JOIN event_observations o USING (event_id)
WHERE e.event_id = $1;
