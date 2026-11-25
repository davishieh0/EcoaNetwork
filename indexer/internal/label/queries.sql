-- The queries of the labels. `make sqlc` turns each into a function of the
-- generated package, which projector.go and repository.go use.

-- name: ChainOfVersion :one
-- The chain a version of a review belongs to.
SELECT chain_root FROM reviews WHERE event_id = $1;

-- name: IsResponse :one
-- Whether an event is a response.
SELECT EXISTS (SELECT 1 FROM responses WHERE event_id = $1);

-- name: IsProposal :one
-- Whether an event is an entity.updated the indexer holds.
SELECT EXISTS (SELECT 1 FROM entity_proposals WHERE event_id = $1);

-- name: IsProfile :one
-- Whether an event is a profile.updated the indexer holds.
SELECT EXISTS (SELECT 1 FROM events
               WHERE event_id = $1 AND event_type = 'profile.updated');

-- name: SaveLabel :exec
-- Stores one label.
INSERT INTO labels (event_id, target, chain_root, response, proposal,
                    profile, moderator, label, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: RevokeLabel :exec
-- Takes a label back, only by the moderator who published it: the same
-- identity, whichever of its keys signed each.
UPDATE labels SET revoked = true
WHERE event_id = sqlc.arg(label)
  AND COALESCE((SELECT i.identity FROM identities i WHERE i.key = moderator),
               moderator)
    = COALESCE((SELECT i.identity FROM identities i
                WHERE i.key = sqlc.arg(moderator)),
               sqlc.arg(moderator)::text);

-- name: KeysOfFollowed :many
-- Every key of the identities some keys belong to: following one key of a
-- moderator follows all of theirs.
SELECT DISTINCT k::text AS key FROM (
    SELECT j.key AS k
    FROM unnest(sqlc.arg(dids)::text[]) AS d
    JOIN identities i ON i.key = d
    JOIN identities j ON j.identity = i.identity
    UNION
    SELECT d FROM unnest(sqlc.arg(dids)::text[]) AS d
) AS followed;

-- name: ChainLabels :many
-- The labels that count on some chains: not revoked, by a followed moderator.
SELECT chain_root::text AS chain_root, event_id, label, moderator
FROM labels
WHERE chain_root = ANY (sqlc.arg(roots)::text[])
  AND NOT revoked
  AND moderator = ANY (sqlc.arg(moderators)::text[])
ORDER BY event_id;

-- name: ResponseLabels :many
-- The labels that count on some responses.
SELECT response::text AS response, event_id, label, moderator
FROM labels
WHERE response = ANY (sqlc.arg(responses)::text[])
  AND NOT revoked
  AND moderator = ANY (sqlc.arg(moderators)::text[])
ORDER BY event_id;

-- name: IsRemoved :one
-- Whether an event is removed by a followed moderator: labeled itself, or a
-- version of a labeled chain.
SELECT EXISTS (
    SELECT 1 FROM labels l
    WHERE NOT l.revoked
      AND l.moderator = ANY (sqlc.arg(moderators)::text[])
      AND l.label = ANY (sqlc.arg(removing)::text[])
      AND (l.target = sqlc.arg(event_id)
           OR l.response = sqlc.arg(event_id)
           OR l.chain_root = (SELECT r.chain_root FROM reviews r
                              WHERE r.event_id = sqlc.arg(event_id)))
);

-- name: ProfileLabels :many
-- The labels that count on some profile.updated events.
SELECT profile::text AS profile, event_id, label, moderator
FROM labels
WHERE profile = ANY (sqlc.arg(profiles)::text[])
  AND NOT revoked
  AND moderator = ANY (sqlc.arg(moderators)::text[])
ORDER BY event_id;

-- name: EventLabels :many
-- The labels that count on one event: on itself, on the chain it is a
-- version of, on it as a response or as a profile.
SELECT event_id, label, moderator
FROM labels l
WHERE NOT l.revoked
  AND l.moderator = ANY (sqlc.arg(moderators)::text[])
  AND (l.target = sqlc.arg(event_id)
       OR l.response = sqlc.arg(event_id)
       OR l.profile = sqlc.arg(event_id)
       OR l.chain_root = (SELECT r.chain_root FROM reviews r
                          WHERE r.event_id = sqlc.arg(event_id)))
ORDER BY event_id;
