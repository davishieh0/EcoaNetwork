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
