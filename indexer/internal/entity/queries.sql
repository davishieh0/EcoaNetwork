-- The queries of the entities. `make sqlc` turns each into a function of the
-- generated package, which projector.go and repository.go use.

-- name: EnsureEntity :exec
-- Records that an event named an entity, keeping the earliest createdAt.
INSERT INTO entities (entity, first_seen)
VALUES ($1, $2)
ON CONFLICT (entity) DO UPDATE
SET first_seen = LEAST(entities.first_seen, EXCLUDED.first_seen);

-- name: SaveProposal :exec
-- Stores one entity.updated.
INSERT INTO entity_proposals (event_id, entity, author, created_at, name,
                              category, description, descriptor, search_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: EntityExists :one
-- Whether the indexer saw an entity, and how many proposals it has.
SELECT EXISTS (SELECT 1 FROM entities n WHERE n.entity = sqlc.arg(id))::boolean
           AS found,
       (SELECT count(*) FROM entity_proposals p
        WHERE p.entity = sqlc.arg(id))::integer AS proposals;

-- name: ProposalEnvelopes :many
-- The proposals still in the contest for some entities, as signed: those no
-- followed moderator labeled.
SELECT p.entity, e.envelope
FROM entity_proposals p
JOIN events e ON e.event_id = p.event_id
WHERE p.entity = ANY (sqlc.arg(entities)::text[])
  AND NOT EXISTS (
      SELECT 1 FROM labels l
      WHERE l.proposal = p.event_id
        AND NOT l.revoked
        AND l.moderator = ANY (sqlc.arg(moderators)::text[]))
ORDER BY p.created_at, p.event_id;

-- name: FirstDescriptor :one
-- The descriptor of an entity's earliest proposal that carries one.
SELECT descriptor::text AS descriptor
FROM entity_proposals
WHERE entity = $1 AND descriptor IS NOT NULL
ORDER BY created_at, event_id
LIMIT 1;

-- name: SearchEntities :many
-- Entities with a proposal whose name holds the text, newest first, after
-- the cursor when there is one; every entity when the text is empty.
SELECT n.entity, n.first_seen
FROM entities n
WHERE (sqlc.arg(query)::text = ''
       OR EXISTS (
           SELECT 1 FROM entity_proposals p
           WHERE p.entity = n.entity
             AND p.search_name LIKE '%' || sqlc.arg(query)::text || '%'
             AND NOT EXISTS (
                 SELECT 1 FROM labels l
                 WHERE l.proposal = p.event_id
                   AND NOT l.revoked
                   AND l.moderator = ANY (sqlc.arg(moderators)::text[]))))
  -- a category: some proposal still in the contest says it; whether it is
  -- the one that holds is checked when the answer is worked out
  AND (sqlc.narg(category)::text IS NULL
       OR EXISTS (
           SELECT 1 FROM entity_proposals p
           WHERE p.entity = n.entity
             AND p.category = sqlc.narg(category)::text
             AND NOT EXISTS (
                 SELECT 1 FROM labels l
                 WHERE l.proposal = p.event_id
                   AND NOT l.revoked
                   AND l.moderator = ANY (sqlc.arg(moderators)::text[]))))
  AND (sqlc.narg(after_seen)::timestamptz IS NULL
       OR (n.first_seen, n.entity)
          < (sqlc.narg(after_seen)::timestamptz, sqlc.narg(after_entity)::text))
ORDER BY n.first_seen DESC, n.entity DESC
LIMIT sqlc.arg(row_limit);
