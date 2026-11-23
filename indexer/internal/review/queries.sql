-- The queries of the reviews. `make sqlc` turns each into a function of the
-- generated package, which repository.go wraps.

-- name: HeadsByEntity :many
-- The head of each visible chain of an entity, newest original first, after
-- the cursor when there is one. A chain a followed moderator labeled with one
-- of the hiding labels is not visible.
SELECT c.chain_root,
       c.created_at AS chain_created_at,
       c.head::text AS head,
       e.envelope,
       o.received_at,
       (SELECT count(*) FROM reviews v WHERE v.chain_root = c.chain_root)
           AS versions,
       COALESCE(t.endorsements, 0)::integer AS endorsements,
       COALESCE(t.disputes, 0)::integer AS disputes
FROM review_chains c
JOIN events e ON e.event_id = c.head
LEFT JOIN event_observations o ON o.event_id = c.head
LEFT JOIN version_votes t ON t.version = c.head
WHERE c.entity = sqlc.arg(entity)
  AND NOT c.withdrawn
  AND NOT c.truncated
  AND NOT EXISTS (
      SELECT 1 FROM labels l
      WHERE l.chain_root = c.chain_root
        AND NOT l.revoked
        AND l.moderator = ANY (sqlc.arg(moderators)::text[])
        AND l.label = ANY (sqlc.arg(hiding)::text[]))
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
       o.received_at,
       COALESCE(t.endorsements, 0)::integer AS endorsements,
       COALESCE(t.disputes, 0)::integer AS disputes
FROM reviews r
JOIN review_chains c ON c.chain_root = r.chain_root
JOIN events e ON e.event_id = r.event_id
LEFT JOIN event_observations o ON o.event_id = r.event_id
LEFT JOIN version_votes t ON t.version = r.event_id
WHERE r.event_id = $1;

-- name: ChainVersions :many
-- Every version of a chain, forks included, oldest first.
SELECT r.event_id,
       e.envelope,
       o.received_at,
       COALESCE(t.endorsements, 0)::integer AS endorsements,
       COALESCE(t.disputes, 0)::integer AS disputes
FROM reviews r
JOIN events e ON e.event_id = r.event_id
LEFT JOIN event_observations o ON o.event_id = r.event_id
LEFT JOIN version_votes t ON t.version = r.event_id
WHERE r.chain_root = $1
ORDER BY r.created_at, r.event_id;

-- name: SaveVersion :exec
-- Stores one version of a review.
INSERT INTO reviews (event_id, chain_root, replaces, author, entity, rating,
                     title, body, lang, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: SaveChain :exec
-- Opens the chain a review.created starts, with the review as its head.
INSERT INTO review_chains (chain_root, entity, head, created_at)
VALUES (sqlc.arg(chain_root), sqlc.arg(entity), sqlc.arg(chain_root),
        sqlc.arg(created_at));

-- name: PlaceOf :one
-- The chain a version belongs to, and the entity its root reviews.
SELECT chain_root, entity FROM reviews WHERE event_id = $1;

-- name: SaveWithdrawal :exec
-- Stores one review.withdrawn.
INSERT INTO review_withdrawals (event_id, chain_root, target)
VALUES ($1, $2, $3);

-- name: ChainEnvelopes :many
-- Every event that decides a chain: its versions and its withdrawals.
SELECT e.envelope
FROM reviews r JOIN events e ON e.event_id = r.event_id
WHERE r.chain_root = $1
UNION ALL
SELECT e.envelope
FROM review_withdrawals w JOIN events e ON e.event_id = w.event_id
WHERE w.chain_root = $1;

-- name: UpdateChain :exec
-- Records what the rules resolved for a chain.
UPDATE review_chains
SET head = sqlc.narg(head),
    withdrawn = sqlc.arg(withdrawn),
    truncated = sqlc.arg(truncated)
WHERE chain_root = sqlc.arg(chain_root);
