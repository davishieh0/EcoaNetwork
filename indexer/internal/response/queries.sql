-- The queries of the responses. `make sqlc` turns each into a function of
-- the generated package, which repository.go wraps.

-- name: PlaceOfTarget :one
-- Where a response lands: the chain of what it answers, the version that
-- conversation answered, how deep what it answers is (0 for a version), and
-- where what it answers sorts (null for a version).
SELECT v.chain_root, v.event_id AS responded_version, 0::integer AS depth,
       NULL::timestamptz AS sort_at
FROM reviews v WHERE v.event_id = sqlc.arg(target)
UNION ALL
SELECT p.chain_root, p.responded_version, p.depth, p.sort_at
FROM responses p WHERE p.event_id = sqlc.arg(target);

-- name: SaveResponse :exec
-- Stores one response.
INSERT INTO responses (event_id, chain_root, target, responded_version,
                       depth, author, body, lang, created_at, sort_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: ResponsesByChain :many
-- A chain's responses in the order of the conversation, after the cursor
-- when there is one, with the chain's current head. A response a followed
-- moderator labeled with one of the hiding labels is left out.
SELECT r.event_id,
       r.target,
       r.responded_version,
       r.depth,
       r.sort_at,
       COALESCE(c.head, '')::text AS head,
       e.envelope,
       o.received_at
FROM responses r
JOIN review_chains c ON c.chain_root = r.chain_root
JOIN events e ON e.event_id = r.event_id
LEFT JOIN event_observations o ON o.event_id = r.event_id
WHERE r.chain_root = sqlc.arg(chain_root)
  AND NOT EXISTS (
      SELECT 1 FROM labels l
      WHERE l.response = r.event_id
        AND NOT l.revoked
        AND l.moderator = ANY (sqlc.arg(moderators)::text[])
        AND l.label = ANY (sqlc.arg(hiding)::text[]))
  AND (sqlc.narg(after_sort_at)::timestamptz IS NULL
       OR (r.sort_at, r.event_id)
          > (sqlc.narg(after_sort_at)::timestamptz,
             sqlc.narg(after_id)::text))
ORDER BY r.sort_at, r.event_id
LIMIT sqlc.arg(row_limit);
