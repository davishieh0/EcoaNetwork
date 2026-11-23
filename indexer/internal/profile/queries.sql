-- The queries of the profiles. `make sqlc` turns each into a function of the
-- generated package, which projector.go and repository.go use.

-- name: CurrentEnvelope :one
-- The profile.updated that holds now for an identity, as signed.
SELECT e.envelope
FROM profiles p JOIN events e ON e.event_id = p.event_id
WHERE p.identity = $1;

-- name: SaveProfile :exec
-- Makes a profile.updated the identity's current profile, replacing every
-- field of the one before.
INSERT INTO profiles (identity, event_id, display_name, bio, lang, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (identity) DO UPDATE
SET event_id = EXCLUDED.event_id,
    display_name = EXCLUDED.display_name,
    bio = EXCLUDED.bio,
    lang = EXCLUDED.lang,
    created_at = EXCLUDED.created_at;

-- name: ProfilesOf :many
-- The current profiles of some identities.
SELECT p.identity, p.event_id, e.envelope, o.received_at
FROM profiles p
JOIN events e ON e.event_id = p.event_id
LEFT JOIN event_observations o ON o.event_id = p.event_id
WHERE p.identity = ANY (sqlc.arg(identities)::text[]);

-- name: IdentityOf :one
-- The identity a key belongs to: itself when it never rotated.
SELECT COALESCE((SELECT i.identity FROM identities i WHERE i.key = $1),
                $1)::text AS identity;

-- name: IdentitiesOf :many
-- The identity of each of some keys.
SELECT d::text AS did, COALESCE(i.identity, d)::text AS identity
FROM unnest(sqlc.arg(dids)::text[]) AS d
LEFT JOIN identities i ON i.key = d;

-- name: KeysOf :many
-- The keys of an identity: none when the name is a key that now belongs to
-- another identity.
SELECT i.key::text AS key FROM identities i WHERE i.identity = sqlc.arg(identity)
UNION
SELECT sqlc.arg(identity)::text
WHERE NOT EXISTS (SELECT 1 FROM identities j
                  WHERE j.key = sqlc.arg(identity)
                    AND j.identity <> sqlc.arg(identity));

-- name: ProfileEnvelopesBy :many
-- Every profile.updated some keys signed, as signed.
SELECT envelope FROM events
WHERE event_type = 'profile.updated' AND author = ANY (sqlc.arg(keys)::text[]);

-- name: DeleteProfile :exec
-- Forgets the profile of a name that is no identity of its own any more.
DELETE FROM profiles WHERE identity = $1;
