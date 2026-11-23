-- The queries of the identities. `make sqlc` turns each into a function of
-- the generated package, which projector.go and repository.go use.

-- name: SaveRotation :exec
-- Stores one identity.key_rotated.
INSERT INTO key_rotations (event_id, key, successor, created_at)
VALUES ($1, $2, $3, $4);

-- name: SaveRevocation :exec
-- Stores one identity.key_revoked.
INSERT INTO key_revocations (event_id, key, reason, created_at)
VALUES ($1, $2, $3, $4);

-- name: AllRotations :many
-- Every rotation, which the rule works the identities out from.
SELECT event_id, key, successor, created_at FROM key_rotations;

-- name: AllAssignments :many
-- Which identity each key that rotated belongs to, as last worked out.
SELECT key, identity, position, contested FROM identities;

-- name: ClearAssignments :exec
-- Forgets every assignment, before writing them again.
DELETE FROM identities;

-- name: SaveAssignment :exec
-- Records which identity a key belongs to.
INSERT INTO identities (key, identity, position, contested)
VALUES ($1, $2, $3, $4);

-- name: AssignmentOf :one
-- Which identity a key belongs to; no row for a key that is its own.
SELECT identity, position, contested FROM identities WHERE key = $1;

-- name: KeysOfIdentity :many
-- Every key of an identity, in the order of its chain.
SELECT key, position, contested FROM identities
WHERE identity = $1
ORDER BY position;

-- name: RevocationsOf :many
-- The revocations of some keys.
SELECT event_id, key, reason FROM key_revocations
WHERE key = ANY (sqlc.arg(keys)::text[])
ORDER BY created_at, event_id;
