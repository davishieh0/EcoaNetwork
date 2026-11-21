-- The queries of the reviews. `make sqlc` turns each into a function of the
-- generated package, which repository.go wraps.

-- name: HeadsByEntity :many
-- The head of each visible chain of an entity, newest original first, after
-- the cursor when there is one.
SELECT c.chain_root,
       c.created_at AS chain_created_at,
       c.head::text AS head,
       e.envelope,
       o.received_at,
       (SELECT count(*) FROM reviews v WHERE v.chain_root = c.chain_root)
           AS versions
FROM review_chains c
JOIN events e ON e.event_id = c.head
LEFT JOIN event_observations o ON o.event_id = c.head
WHERE c.entity = sqlc.arg(entity)
  AND NOT c.withdrawn
  AND NOT c.truncated
  AND (sqlc.narg(after_created_at)::timestamptz IS NULL
       OR (c.created_at, c.chain_root)
          < (sqlc.narg(after_created_at)::timestamptz,
             sqlc.narg(after_root)::text))
ORDER BY c.created_at DESC, c.chain_root DESC
LIMIT sqlc.arg(row_limit);

-- name: ReviewByID :one
-- One version, with the state of its chain.
SELECT r.event_id,
       r.chain_root,
       c.head,
       c.withdrawn,
       c.truncated,
       e.envelope,
       o.received_at
FROM reviews r
JOIN review_chains c ON c.chain_root = r.chain_root
JOIN events e ON e.event_id = r.event_id
LEFT JOIN event_observations o ON o.event_id = r.event_id
WHERE r.event_id = $1;

-- name: ChainVersions :many
-- Every version of a chain, forks included, oldest first.
SELECT r.event_id,
       e.envelope,
       o.received_at
FROM reviews r
JOIN events e ON e.event_id = r.event_id
LEFT JOIN event_observations o ON o.event_id = r.event_id
WHERE r.chain_root = $1
ORDER BY r.created_at, r.event_id;
