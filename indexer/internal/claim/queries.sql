-- The queries of the claims. `make sqlc` turns each into a function of the
-- generated package, which projector.go and repository.go use.

-- name: SaveClaim :exec
-- Stores one claim.
INSERT INTO claims (event_id, claim_type, issuer, subject, entity,
                    evidence_ref, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: RevokeClaim :exec
-- Takes a claim back, only by its issuer: the same identity, whichever of
-- its keys signed each.
UPDATE claims SET revoked = true
WHERE event_id = sqlc.arg(claim)
  AND COALESCE((SELECT i.identity FROM identities i WHERE i.key = issuer),
               issuer)
    = COALESCE((SELECT i.identity FROM identities i
                WHERE i.key = sqlc.arg(issuer)),
               sqlc.arg(issuer)::text);

-- name: DueChecks :many
-- The domain claims whose lookup is due: never checked, verified longer ago
-- than recheck, or failed longer ago than retry. Only the ones that could
-- count: self-signed, live, about a domain.
SELECT c.event_id, c.subject, c.entity::text AS entity
FROM claims c
LEFT JOIN claim_checks k USING (event_id)
WHERE c.claim_type = 'domain'
  AND c.issuer = c.subject
  AND NOT c.revoked
  AND c.expires_at > sqlc.arg(now)
  AND c.entity LIKE 'domain:%'
  AND (k.event_id IS NULL
       OR (k.verified AND k.checked_at < sqlc.arg(recheck_before))
       OR (NOT k.verified AND k.checked_at < sqlc.arg(retry_before)))
ORDER BY k.checked_at NULLS FIRST, c.event_id
LIMIT sqlc.arg(most);

-- name: SaveCheck :exec
-- Records a lookup. A pass keeps the time it first passed without a gap; a
-- failure forgets it, so a representative who lapsed starts over.
INSERT INTO claim_checks (event_id, verified, dnssec, checked_at,
                          verified_since, detail)
VALUES (sqlc.arg(event_id), sqlc.arg(verified)::boolean, sqlc.arg(dnssec),
        sqlc.arg(checked_at)::timestamptz,
        CASE WHEN sqlc.arg(verified)::boolean
             THEN sqlc.arg(checked_at)::timestamptz END,
        sqlc.arg(detail))
ON CONFLICT (event_id) DO UPDATE
SET verified       = EXCLUDED.verified,
    dnssec         = EXCLUDED.dnssec,
    checked_at     = EXCLUDED.checked_at,
    verified_since = CASE
        WHEN NOT EXCLUDED.verified THEN NULL
        WHEN claim_checks.verified THEN claim_checks.verified_since
        ELSE EXCLUDED.checked_at
    END,
    detail         = EXCLUDED.detail;

-- name: RecognizedOf :many
-- The claims that count for some subjects: live and not revoked, a domain
-- claim self-signed and verified by a lookup not older than fresh_after, a
-- human claim issued by a verifier this indexer follows.
SELECT c.event_id, c.claim_type, c.subject, c.entity, c.expires_at,
       k.verified_since, COALESCE(k.dnssec, false) AS dnssec
FROM claims c
LEFT JOIN claim_checks k USING (event_id)
WHERE c.subject = ANY (sqlc.arg(subjects)::text[])
  AND NOT c.revoked
  AND c.expires_at > sqlc.arg(now)
  AND ((c.claim_type = 'domain' AND c.issuer = c.subject
        AND k.verified AND k.checked_at > sqlc.arg(fresh_after))
       OR (c.claim_type = 'human'
           AND c.issuer = ANY (sqlc.arg(verifiers)::text[])))
ORDER BY c.subject, c.claim_type, c.event_id;

-- name: RepresentativesOf :many
-- The identities holding a domain claim that counts on some entities.
SELECT c.entity::text AS entity, c.subject, c.event_id, k.verified_since
FROM claims c
JOIN claim_checks k USING (event_id)
WHERE c.entity = ANY (sqlc.arg(entities)::text[])
  AND c.claim_type = 'domain'
  AND c.issuer = c.subject
  AND NOT c.revoked
  AND c.expires_at > sqlc.arg(now)
  AND k.verified
  AND k.checked_at > sqlc.arg(fresh_after)
ORDER BY c.entity, k.verified_since, c.event_id;

-- name: EntityOfChain :one
-- The entity a review chain is about.
SELECT entity FROM review_chains WHERE chain_root = $1;
